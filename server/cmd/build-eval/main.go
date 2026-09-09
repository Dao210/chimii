// build-eval runs offline Build experiments without loading application config.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/chimii-ai/chimii/server/internal/build"
	"github.com/chimii-ai/chimii/server/internal/buildeval"
)

func main() {
	output := flag.String("output", "", "JSON report path; stdout when empty")
	family := flag.String("family", "", "comma-separated case families; all by default")
	corpus := flag.String("corpus", "baseline", "fixed corpus: baseline or seams")
	candidatePath := flag.String("candidates", "", "optional versioned BrickGPT/dataset candidate JSON file")
	catalogName := flag.String("catalog", "starter", "catalog: starter or long-bricks (four existing manifest parts)")
	archive := flag.String("archive", "", "pinned LDraw complete.zip; required for long-bricks")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	catalog := build.StarterCatalog
	var assets []buildeval.CatalogAsset
	var sourceHash string
	switch *catalogName {
	case "starter":
	case "long-bricks":
		var err error
		catalog, assets, err = buildeval.LongBrickCatalog(*archive)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown evaluation catalog")
		os.Exit(2)
	}
	cases := buildeval.BaselineCases()
	switch *corpus {
	case "baseline":
	case "seams":
		if *candidatePath != "" {
			fmt.Fprintln(os.Stderr, "seams corpus cannot be combined with candidates")
			os.Exit(2)
		}
		cases = buildeval.SeamCases()
	default:
		fmt.Fprintln(os.Stderr, "unknown evaluation corpus")
		os.Exit(2)
	}
	if *candidatePath != "" {
		f, err := os.Open(*candidatePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || info.Size() > 4*1024*1024 {
			fmt.Fprintln(os.Stderr, "candidate input cannot exceed 4 MiB")
			os.Exit(2)
		}
		dec := json.NewDecoder(io.LimitReader(f, 4*1024*1024))
		dec.DisallowUnknownFields()
		var batch buildeval.CandidateBatch
		err = dec.Decode(&batch)
		if err == nil {
			var extra any
			if dec.Decode(&extra) != io.EOF {
				err = fmt.Errorf("trailing data in candidate file")
			}
		}
		if err == nil {
			cases, err = buildeval.ImportCandidates(batch, catalog)
			sourceHash = buildeval.Hash(batch)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	if *family != "" {
		selected := cases[:0]
		for _, c := range cases {
			for _, f := range strings.Split(*family, ",") {
				if c.Family == f {
					selected = append(selected, c)
					break
				}
			}
		}
		cases = selected
	}
	if len(cases) == 0 {
		fmt.Fprintln(os.Stderr, "no evaluation cases selected")
		os.Exit(2)
	}
	report := buildeval.Run(ctx, cases, catalog)
	report.CatalogAssets, report.SourceHash = assets, sourceHash
	if *corpus == "seams" {
		report.CorpusVersion = buildeval.SeamCorpusVersion
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err == nil {
		data = append(data, '\n')
		if *output == "" {
			_, err = os.Stdout.Write(data)
		} else {
			err = os.WriteFile(*output, data, 0o644)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	fmt.Fprintf(os.Stderr, "%d cases: %d accepted, %d rejected, %d observed, %d contract failures\n", report.Summary.Total, report.Summary.Accepted, report.Summary.Rejected, report.Summary.Observed, report.Summary.ContractFailed)
	if report.Summary.ContractFailed > 0 {
		os.Exit(1)
	}
}
