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

	"github.com/chimii-ai/chimii/server/internal/buildeval"
)

func main() {
	output := flag.String("output", "", "JSON report path; stdout when empty")
	family := flag.String("family", "", "comma-separated case families; all by default")
	candidatePath := flag.String("candidates", "", "optional versioned BrickGPT/dataset candidate JSON file")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	cases := buildeval.BaselineCases()
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
			cases, err = buildeval.ImportCandidates(batch)
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
	report := buildeval.Run(ctx, cases)
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
