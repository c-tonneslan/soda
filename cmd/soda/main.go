// soda is a CLI for working with Socrata-based open data portals.
//
// Usage examples:
//
//	soda portals
//	soda ls nyc --limit 25
//	soda info nyc erm2-nwe9
//	soda pull nyc erm2-nwe9 --limit 1000 -o nyc311.json
//	soda search "affordable housing"
//	soda search permits --portal nyc
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/spf13/cobra"

	"github.com/c-tonneslan/soda/internal/output"
	"github.com/c-tonneslan/soda/internal/portals"
	"github.com/c-tonneslan/soda/internal/socrata"
)

var version = "0.1.0"

func main() {
	if err := newRoot().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "soda",
		Short: "A CLI for Socrata-based open data portals.",
		Long: `soda is a CLI for working with Socrata-based open data portals.

Hits Socrata's per-portal SODA endpoints and the global Discovery / catalog
API. Ships with a registry of known US municipal and state portals; adding a
new one is a one-line config change.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.AddCommand(newPortalsCmd(), newLsCmd(), newInfoCmd(), newPullCmd(), newSearchCmd())
	return root
}

func ctxFromCmd(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(cmd.Context(), os.Interrupt)
}

func newClient() *socrata.Client {
	c := socrata.New()
	if t := os.Getenv("SODA_APP_TOKEN"); t != "" {
		c.AppToken = t
	}
	return c
}

// ----- portals

func newPortalsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "portals",
		Short: "List the open-data portals soda knows about.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			rows := [][]string{{"SLUG", "NAME", "HOST"}}
			for _, p := range portals.All() {
				rows = append(rows, []string{p.Slug, p.Name, p.Host})
			}
			return output.Table(cmd.OutOrStdout(), rows)
		},
	}
}

// ----- ls

func newLsCmd() *cobra.Command {
	var limit, offset int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "ls <portal>",
		Short: "List datasets in a portal.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := portals.Get(args[0])
			if err != nil {
				return err
			}
			ctx, cancel := ctxFromCmd(cmd)
			defer cancel()
			datasets, err := newClient().Catalog(ctx, p.Host, limit, offset)
			if err != nil {
				return err
			}
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), datasets)
			}
			rows := [][]string{{"ID", "UPDATED", "NAME"}}
			for _, d := range datasets {
				rows = append(rows, []string{d.ID, short(d.Updated), trunc(d.Name, 70)})
			}
			return output.Table(cmd.OutOrStdout(), rows)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 25, "Max datasets to list (cap 100)")
	cmd.Flags().IntVar(&offset, "offset", 0, "Pagination offset")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Emit JSON instead of a table")
	return cmd
}

// ----- info

func newInfoCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "info <portal> <dataset-id>",
		Short: "Show metadata and schema for a dataset.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := portals.Get(args[0])
			if err != nil {
				return err
			}
			ctx, cancel := ctxFromCmd(cmd)
			defer cancel()
			schema, err := newClient().Info(ctx, p.Host, args[1])
			if err != nil {
				return err
			}
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), schema)
			}
			w := cmd.OutOrStdout()
			fmt.Fprintln(w, schema.Name)
			if schema.Attribution != "" {
				fmt.Fprintln(w, "by", schema.Attribution)
			}
			fmt.Fprintln(w, "updated", short(schema.Updated))
			if schema.Description != "" {
				fmt.Fprintln(w)
				fmt.Fprintln(w, trunc(schema.Description, 400))
			}
			fmt.Fprintln(w)
			rows := [][]string{{"FIELD", "TYPE", "LABEL"}}
			for _, c := range schema.Columns {
				rows = append(rows, []string{c.FieldName, c.DataType, c.Name})
			}
			return output.Table(w, rows)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Emit JSON instead of a table")
	return cmd
}

// ----- pull

func newPullCmd() *cobra.Command {
	var asCSV bool
	var limit, offset int
	var where, order, selectExpr, outPath string
	cmd := &cobra.Command{
		Use:   "pull <portal> <dataset-id>",
		Short: "Download dataset rows.",
		Long: `Download dataset rows.

The default output is JSON; --csv switches to CSV. Pass SoQL clauses via
--where, --order, and --select for filtering and shaping. Without --limit,
Socrata returns its default of 1000 rows per call.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := portals.Get(args[0])
			if err != nil {
				return err
			}
			format := socrata.FormatJSON
			if asCSV {
				format = socrata.FormatCSV
			}
			ctx, cancel := ctxFromCmd(cmd)
			defer cancel()
			body, err := newClient().Rows(ctx, p.Host, args[1], socrata.PullOptions{
				Format: format, Limit: limit, Offset: offset,
				Where: where, Order: order, Select: selectExpr,
			})
			if err != nil {
				return err
			}
			defer body.Close()
			dest := cmd.OutOrStdout()
			if outPath != "" {
				f, err := os.Create(outPath)
				if err != nil {
					return err
				}
				defer f.Close()
				dest = f
			}
			n, err := io.Copy(dest, body)
			if err != nil {
				return err
			}
			if outPath != "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "wrote %d bytes to %s\n", n, outPath)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asCSV, "csv", false, "Emit CSV (default JSON)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max rows (0 = Socrata default of 1000)")
	cmd.Flags().IntVar(&offset, "offset", 0, "Pagination offset")
	cmd.Flags().StringVar(&where, "where", "", "SoQL $where clause")
	cmd.Flags().StringVar(&order, "order", "", "SoQL $order clause")
	cmd.Flags().StringVar(&selectExpr, "select", "", "SoQL $select clause")
	cmd.Flags().StringVarP(&outPath, "output", "o", "", "Write to file instead of stdout")
	return cmd
}

// ----- search

func newSearchCmd() *cobra.Command {
	var limit int
	var portalSlug string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Socrata datasets across one or every portal.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			var domains []string
			if portalSlug != "" {
				p, err := portals.Get(portalSlug)
				if err != nil {
					return err
				}
				domains = []string{p.Host}
			}
			ctx, cancel := ctxFromCmd(cmd)
			defer cancel()
			hits, err := newClient().Search(ctx, query, domains, limit)
			if err != nil {
				return err
			}
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), hits)
			}
			rows := [][]string{{"ID", "DOMAIN", "UPDATED", "NAME"}}
			for _, h := range hits {
				rows = append(rows, []string{h.ID, h.Domain, short(h.Updated), trunc(h.Name, 60)})
			}
			return output.Table(cmd.OutOrStdout(), rows)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Max hits (cap 100)")
	cmd.Flags().StringVar(&portalSlug, "portal", "", "Restrict to one portal slug")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Emit JSON instead of a table")
	return cmd
}

// ----- helpers

func short(iso string) string {
	if len(iso) >= 10 {
		return iso[:10]
	}
	return iso
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
