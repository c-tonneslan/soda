// soda is a CLI for working with Socrata-based open data portals.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/c-tonneslan/soda/internal/cache"
	"github.com/c-tonneslan/soda/internal/diff"
	"github.com/c-tonneslan/soda/internal/output"
	"github.com/c-tonneslan/soda/internal/portals"
	"github.com/c-tonneslan/soda/internal/socrata"
	"github.com/c-tonneslan/soda/internal/sqlitesink"
)

var version = "0.4.0"

// Global flags applied across every command via PersistentFlags.
type globalFlags struct {
	verbose bool
	useCache bool
}

func main() {
	if err := newRoot().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRoot() *cobra.Command {
	g := &globalFlags{}
	root := &cobra.Command{
		Use:   "soda",
		Short: "A CLI for Socrata-based open data portals.",
		Long: `soda is a CLI for working with Socrata-based open data portals.

Hits Socrata's per-portal SODA endpoints and the global Discovery API. Ships
with a registry of known US/international portals; adding a new one is a
one-line config change.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.PersistentFlags().BoolVar(&g.verbose, "verbose", false, "Log each HTTP request to stderr")
	root.PersistentFlags().BoolVar(&g.useCache, "cache", false,
		"Cache GET responses under ~/.cache/soda (safe for read-only data)")
	root.AddCommand(
		newPortalsCmd(),
		newLsCmd(g),
		newInfoCmd(g),
		newPullCmd(g),
		newSearchCmd(g),
		newStatsCmd(g),
		newOpenCmd(),
		newWatchCmd(g),
		newDiffCmd(),
	)
	return root
}

func ctxFromCmd(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(cmd.Context(), os.Interrupt)
}

// newClient builds an http client wired up to honor --cache and --verbose.
func newClient(g *globalFlags) *socrata.Client {
	transport := http.DefaultTransport
	if g.useCache {
		transport = cache.New("", transport)
	}
	if g.verbose {
		transport = &loggingTransport{Inner: transport}
	}
	c := socrata.New()
	c.HTTP.Transport = transport
	if t := os.Getenv("SODA_APP_TOKEN"); t != "" {
		c.AppToken = t
	}
	return c
}

type loggingTransport struct{ Inner http.RoundTripper }

func (lt *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Fprintf(os.Stderr, "GET %s\n", req.URL)
	return lt.Inner.RoundTrip(req)
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

func newLsCmd(g *globalFlags) *cobra.Command {
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
			datasets, err := newClient(g).Catalog(ctx, p.Host, limit, offset)
			if err != nil {
				return hint(err)
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

func newInfoCmd(g *globalFlags) *cobra.Command {
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
			schema, err := newClient(g).Info(ctx, p.Host, args[1])
			if err != nil {
				return hint(err)
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

func newPullCmd(g *globalFlags) *cobra.Command {
	var format string
	var limit, offset, pageSize int
	var all bool
	var where, order, selectExpr, outPath, dbPath string
	cmd := &cobra.Command{
		Use:   "pull <portal> <dataset-id>",
		Short: "Download dataset rows.",
		Long: `Download dataset rows.

Default output is JSON. --format=csv emits CSV, --format=ndjson emits one
JSON object per line. --to <file.db> writes into a SQLite database (one
table per dataset, named after the four-by-four). With --all, soda
auto-paginates through the entire dataset; without it you get one page.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := portals.Get(args[0])
			if err != nil {
				return err
			}
			ctx, cancel := ctxFromCmd(cmd)
			defer cancel()
			client := newClient(g)

			fmtName := strings.ToLower(format)
			if dbPath != "" && (fmtName != "" && fmtName != "json") {
				return fmt.Errorf("--to <db> is only compatible with --format=json (the default)")
			}
			if all {
				if fmtName == "csv" {
					return fmt.Errorf("--all is not compatible with --format=csv; pipe ndjson instead")
				}
				return pullAll(ctx, client, p, args[1], pullParams{
					where: where, order: order, selectExpr: selectExpr,
					pageSize: pageSize, format: fmtName, outPath: outPath, dbPath: dbPath,
					stderr: cmd.ErrOrStderr(),
				})
			}

			// Single-page path
			if dbPath != "" {
				return pullToDB(ctx, client, p, args[1], dbPath, pullParams{
					where: where, order: order, selectExpr: selectExpr,
					limit: limit, offset: offset, stderr: cmd.ErrOrStderr(),
				})
			}
			f := socrata.FormatJSON
			if fmtName == "csv" {
				f = socrata.FormatCSV
			}
			body, err := client.Rows(ctx, p.Host, args[1], socrata.PullOptions{
				Format: f, Limit: limit, Offset: offset,
				Where: where, Order: order, Select: selectExpr,
			})
			if err != nil {
				return hint(err)
			}
			defer body.Close()
			dest, closer, err := openOutput(outPath, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			defer closer()
			if fmtName == "ndjson" {
				return convertJSONToNDJSON(body, dest)
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
	cmd.Flags().StringVar(&format, "format", "json", "Output format: json, csv, or ndjson")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max rows per request (0 = Socrata's default of 1000)")
	cmd.Flags().IntVar(&offset, "offset", 0, "Pagination offset")
	cmd.Flags().IntVar(&pageSize, "page-size", 50000, "Rows per request when paging with --all (cap 50000)")
	cmd.Flags().BoolVar(&all, "all", false, "Auto-paginate the entire dataset (or filtered subset)")
	cmd.Flags().StringVar(&where, "where", "", "SoQL $where clause")
	cmd.Flags().StringVar(&order, "order", "", "SoQL $order clause (recommended with --all)")
	cmd.Flags().StringVar(&selectExpr, "select", "", "SoQL $select clause")
	cmd.Flags().StringVarP(&outPath, "output", "o", "", "Write to file instead of stdout")
	cmd.Flags().StringVar(&dbPath, "to", "", "Write into a SQLite database at this path")

	// CSV shortcut: --csv is equivalent to --format=csv
	var csv bool
	cmd.Flags().BoolVar(&csv, "csv", false, "Shortcut for --format=csv")
	cmd.PreRunE = func(cmd *cobra.Command, _ []string) error {
		if csv {
			format = "csv"
		}
		return nil
	}
	return cmd
}

// ----- search

func newSearchCmd(g *globalFlags) *cobra.Command {
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
			hits, err := newClient(g).Search(ctx, query, domains, limit)
			if err != nil {
				return hint(err)
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

// ----- stats

func newStatsCmd(g *globalFlags) *cobra.Command {
	var where string
	cmd := &cobra.Command{
		Use:   "stats <portal> <dataset-id>",
		Short: "Show row count and date range for a dataset without downloading it.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := portals.Get(args[0])
			if err != nil {
				return err
			}
			ctx, cancel := ctxFromCmd(cmd)
			defer cancel()
			client := newClient(g)
			schema, err := client.Info(ctx, p.Host, args[1])
			if err != nil {
				return hint(err)
			}
			n, err := client.Count(ctx, p.Host, args[1], where)
			if err != nil {
				return hint(err)
			}
			w := cmd.OutOrStdout()
			fmt.Fprintln(w, schema.Name)
			fmt.Fprintf(w, "rows:     %d\n", n)
			fmt.Fprintf(w, "columns:  %d\n", len(schema.Columns))
			fmt.Fprintf(w, "updated:  %s\n", short(schema.Updated))

			// Find the first date-like column and report its min/max.
			for _, c := range schema.Columns {
				if isDateType(c.DataType) {
					body, err := client.Rows(ctx, p.Host, args[1], socrata.PullOptions{
						Format: socrata.FormatJSON,
						Select: fmt.Sprintf("min(%s),max(%s)", c.FieldName, c.FieldName),
						Where:  where,
					})
					if err != nil {
						return nil // best-effort
					}
					defer body.Close()
					var out []map[string]string
					if err := json.NewDecoder(body).Decode(&out); err == nil && len(out) > 0 {
						minKey := "min_" + c.FieldName
						maxKey := "max_" + c.FieldName
						fmt.Fprintf(w, "earliest %s: %s\n", c.FieldName, out[0][minKey])
						fmt.Fprintf(w, "latest %s:   %s\n", c.FieldName, out[0][maxKey])
					}
					break
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&where, "where", "", "Restrict the row count + date range to a SoQL $where clause")
	return cmd
}

// ----- open

func newOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open <portal> <dataset-id>",
		Short: "Open the dataset's web page in your default browser.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := portals.Get(args[0])
			if err != nil {
				return err
			}
			url := fmt.Sprintf("https://%s/d/%s", p.Host, args[1])
			fmt.Fprintln(cmd.OutOrStdout(), url)
			return openInBrowser(url)
		},
	}
}

// ----- watch

func newWatchCmd(g *globalFlags) *cobra.Command {
	var interval time.Duration
	var sinceCol string
	var stateFile string
	var once bool
	cmd := &cobra.Command{
		Use:   "watch <portal> <dataset-id>",
		Short: "Poll a dataset on an interval and emit only new rows.",
		Long: `Poll a dataset on an interval and emit only new rows.

watch reads a high-watermark timestamp (defaults to :updated_at) and only
fetches rows whose value is strictly greater than the last seen mark. The
mark is persisted to a state file so restarts pick up where they left off.

Each new row is emitted as a JSON object on its own line (NDJSON). Pipe to
a logger, a Slack webhook, or another shell script.

Use --once to do a single poll and exit (good for cron). Without it, watch
loops forever on --interval.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := portals.Get(args[0])
			if err != nil {
				return err
			}
			id := args[1]
			if stateFile == "" {
				home, _ := os.UserHomeDir()
				stateFile = fmt.Sprintf("%s/.cache/soda/watch_%s_%s.state",
					home, p.Slug, strings.ReplaceAll(id, "-", "_"))
			}
			if err := os.MkdirAll(stripFile(stateFile), 0o755); err != nil {
				return err
			}
			ctx, cancel := ctxFromCmd(cmd)
			defer cancel()
			client := newClient(g)
			out := cmd.OutOrStdout()
			stderr := cmd.ErrOrStderr()

			// `*` doesn't include `:` metadata fields, so we always select them.
			selectCols := fmt.Sprintf("*,%s", sinceCol)

			poll := func() error {
				mark := readState(stateFile)
				if mark == "" {
					// First run: use the dataset's current max watermark as the
					// starting point so we don't emit history.
					body, err := client.Rows(ctx, p.Host, id, socrata.PullOptions{
						Format: socrata.FormatJSON,
						Select: fmt.Sprintf("max(%s)", sinceCol),
					})
					if err != nil {
						return hint(err)
					}
					defer body.Close()
					var out []map[string]string
					if err := json.NewDecoder(body).Decode(&out); err == nil && len(out) > 0 {
						// Socrata strips the leading `:` from aggregate result keys, so
						// max(:updated_at) comes back under "max_updated_at".
						aggKey := "max_" + strings.TrimPrefix(sinceCol, ":")
						mark = out[0][aggKey]
					}
					if mark != "" {
						_ = writeState(stateFile, mark)
					}
					fmt.Fprintf(stderr, "[%s] initial watermark set to %s\n",
						time.Now().Format(time.RFC3339), mark)
					return nil
				}

				params := socrata.PullOptions{
					Limit:  50000,
					Where:  fmt.Sprintf("%s > '%s'", sinceCol, mark),
					Order:  sinceCol + " ASC",
					Select: selectCols,
				}
				rows, err := client.Pages(ctx, p.Host, id, params)
				if err != nil {
					return hint(err)
				}
				if len(rows) == 0 {
					fmt.Fprintf(stderr, "[%s] no new rows since %s\n",
						time.Now().Format(time.RFC3339), mark)
					return nil
				}
				enc := json.NewEncoder(out)
				var latest string
				for _, r := range rows {
					if v, ok := r[sinceCol]; ok {
						if s := fmt.Sprint(v); s > latest {
							latest = s
						}
					}
					if err := enc.Encode(r); err != nil {
						return err
					}
				}
				if latest != "" {
					if err := writeState(stateFile, latest); err != nil {
						return fmt.Errorf("write state: %w", err)
					}
				}
				fmt.Fprintf(stderr, "[%s] %d new rows, watermark now %s\n",
					time.Now().Format(time.RFC3339), len(rows), latest)
				return nil
			}

			if err := poll(); err != nil {
				return err
			}
			if once {
				return nil
			}
			t := time.NewTicker(interval)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return nil
				case <-t.C:
					if err := poll(); err != nil {
						fmt.Fprintf(stderr, "poll: %v\n", err)
					}
				}
			}
		},
	}
	cmd.Flags().DurationVar(&interval, "interval", 60*time.Second,
		"How often to poll (e.g. 30s, 5m)")
	cmd.Flags().StringVar(&sinceCol, "since-column", ":updated_at",
		"Timestamp column to use as the watermark")
	cmd.Flags().StringVar(&stateFile, "state-file", "",
		"Path to persist the last-seen watermark (default ~/.cache/soda/watch_*.state)")
	cmd.Flags().BoolVar(&once, "once", false, "Poll once and exit")
	return cmd
}

func readState(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func writeState(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func stripFile(path string) string {
	i := strings.LastIndex(path, "/")
	if i < 0 {
		return "."
	}
	return path[:i]
}

// ----- diff

func newDiffCmd() *cobra.Command {
	var keyCol string
	var format string
	cmd := &cobra.Command{
		Use:   "diff <before.json> <after.json>",
		Short: "Compare two snapshots of a Socrata dataset.",
		Long: `Compare two snapshots of a Socrata dataset.

Both files must be JSON arrays of objects (the default output of soda pull).
Rows are identified by their key column (defaults to :id, Socrata's internal
row identifier; override with --key for datasets exposed without :id).

Output is a JSON object with three lists: added, removed, and changed. Pass
--format=summary for a human-readable count.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := readRows(args[0])
			if err != nil {
				return fmt.Errorf("%s: %w", args[0], err)
			}
			b, err := readRows(args[1])
			if err != nil {
				return fmt.Errorf("%s: %w", args[1], err)
			}
			result, err := diff.Compute(a, b, keyCol)
			if err != nil {
				return err
			}
			switch strings.ToLower(format) {
			case "json":
				return output.JSON(cmd.OutOrStdout(), result)
			case "summary", "":
				fmt.Fprintf(cmd.OutOrStdout(),
					"diff %s -> %s (key=%s)\n  added:   %d\n  removed: %d\n  changed: %d\n",
					args[0], args[1], result.Key,
					len(result.Added), len(result.Removed), len(result.Changed))
				if len(result.Changed) > 0 {
					sample := result.Changed
					if len(sample) > 3 {
						sample = sample[:3]
					}
					fmt.Fprintln(cmd.OutOrStdout(), "\nsample changes:")
					for _, c := range sample {
						fields := make([]string, 0, len(c.Changes))
						for _, ch := range c.Changes {
							fields = append(fields, ch.Field)
						}
						sort.Strings(fields)
						fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s\n", c.Key, strings.Join(fields, ", "))
					}
				}
				return nil
			default:
				return fmt.Errorf("unknown --format %q (want json or summary)", format)
			}
		},
	}
	cmd.Flags().StringVar(&keyCol, "key", ":id", "Row identifier column to join on")
	cmd.Flags().StringVar(&format, "format", "summary", "Output format: summary or json")
	return cmd
}

func readRows(path string) ([]diff.Row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var rows []diff.Row
	if err := json.NewDecoder(f).Decode(&rows); err != nil {
		return nil, fmt.Errorf("parse JSON array: %w", err)
	}
	return rows, nil
}

// ----- pull internals

type pullParams struct {
	where, order, selectExpr string
	limit, offset, pageSize  int
	format, outPath, dbPath  string
	stderr                   io.Writer
}

// pullToDB pulls a single page into SQLite.
func pullToDB(ctx context.Context, c *socrata.Client, p portals.Portal,
	id, dbPath string, params pullParams) error {
	schema, err := c.Info(ctx, p.Host, id)
	if err != nil {
		return hint(err)
	}
	sink, err := sqlitesink.Open(dbPath, schema)
	if err != nil {
		return err
	}
	defer sink.Close()
	rows, err := c.Pages(ctx, p.Host, id, socrata.PullOptions{
		Limit: params.limit, Offset: params.offset,
		Where: params.where, Order: params.order, Select: params.selectExpr,
	})
	if err != nil {
		return hint(err)
	}
	n, err := sink.Insert(rows)
	if err != nil {
		return err
	}
	fmt.Fprintf(params.stderr, "wrote %d rows into %s (table %s)\n", n, dbPath, sink.Table)
	return nil
}

// pullAll auto-paginates and writes to the chosen sink.
func pullAll(ctx context.Context, c *socrata.Client, p portals.Portal,
	id string, params pullParams) error {
	pageSize := params.pageSize
	if pageSize <= 0 || pageSize > 50000 {
		pageSize = 50000
	}

	var sink *sqlitesink.Sink
	var ndjsonOut io.Writer
	var jsonRows []map[string]any
	var closer func() error = func() error { return nil }
	asJSONArray := params.format == "" || params.format == "json"

	if params.dbPath != "" {
		schema, err := c.Info(ctx, p.Host, id)
		if err != nil {
			return hint(err)
		}
		s, err := sqlitesink.Open(params.dbPath, schema)
		if err != nil {
			return err
		}
		sink = s
		closer = sink.Close
	} else {
		var w io.Writer = os.Stdout
		if params.outPath != "" {
			f, err := os.Create(params.outPath)
			if err != nil {
				return err
			}
			w = f
			closer = f.Close
		}
		if params.format == "ndjson" {
			ndjsonOut = w
		} else if asJSONArray {
			// Accumulate; we'll write the array at the end so the JSON is valid.
			defer func() {
				_ = output.JSON(w, jsonRows)
				_ = closer()
			}()
		}
	}
	defer closer()

	total := 0
	for offset := 0; ; offset += pageSize {
		rows, err := c.Pages(ctx, p.Host, id, socrata.PullOptions{
			Limit: pageSize, Offset: offset,
			Where: params.where, Order: params.order, Select: params.selectExpr,
		})
		if err != nil {
			return hint(err)
		}
		if len(rows) == 0 {
			break
		}
		switch {
		case sink != nil:
			if _, err := sink.Insert(rows); err != nil {
				return err
			}
		case ndjsonOut != nil:
			enc := json.NewEncoder(ndjsonOut)
			for _, r := range rows {
				if err := enc.Encode(r); err != nil {
					return err
				}
			}
		default:
			jsonRows = append(jsonRows, rows...)
		}
		total += len(rows)
		fmt.Fprintf(params.stderr, "fetched %d rows (offset=%d)\n", total, offset+pageSize)
		if len(rows) < pageSize {
			break
		}
	}
	if sink != nil {
		fmt.Fprintf(params.stderr, "wrote %d rows into %s (table %s)\n", total, params.dbPath, sink.Table)
	}
	return nil
}

// ----- helpers

func openOutput(path string, fallback io.Writer) (io.Writer, func(), error) {
	if path == "" {
		return fallback, func() {}, nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { f.Close() }, nil
}

func convertJSONToNDJSON(body io.Reader, w io.Writer) error {
	var rows []map[string]any
	if err := json.NewDecoder(body).Decode(&rows); err != nil {
		return fmt.Errorf("decode page for ndjson: %w", err)
	}
	enc := json.NewEncoder(w)
	for _, r := range rows {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	return nil
}

// hint wraps a socrata APIError with a friendlier message for known cases.
func hint(err error) error {
	if err == nil {
		return nil
	}
	apiErr, ok := err.(*socrata.APIError)
	if !ok {
		return err
	}
	switch apiErr.Status {
	case http.StatusTooManyRequests:
		if os.Getenv("SODA_APP_TOKEN") == "" {
			return fmt.Errorf("%w\n\nhint: Socrata is rate-limiting you. Get a free App Token "+
				"at the portal you're hitting and export it as SODA_APP_TOKEN", err)
		}
	case http.StatusForbidden:
		return fmt.Errorf("%w\n\nhint: portal rejected the request. Some portals "+
			"require an App Token for any access; export SODA_APP_TOKEN", err)
	case http.StatusNotFound:
		return fmt.Errorf("%w\n\nhint: check the four-by-four ID with `soda search`", err)
	}
	return err
}

func openInBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler"}
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func isDateType(t string) bool {
	t = strings.ToLower(t)
	return strings.Contains(t, "date") || strings.Contains(t, "timestamp")
}

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
