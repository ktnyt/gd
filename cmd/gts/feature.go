package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-gts/gts"
	"github.com/go-gts/gts/flags"
	"github.com/go-gts/gts/seqio"
	"github.com/go-pars/pars"
)

func init() {
	flags.Register("clear", "remove all features from the sequence (excluding source features)", featureClear)
	flags.Register("select", "select features using the given feature selector(s)", featureSelect)
	flags.Register("annotate", "merge features from a feature list file into a sequence", featureAnnotate)
	flags.Register("extract", "extract information from the given sequence", featureExtract)
	flags.Register("seq", "retrieve the sequences referenced by the features", featureSeq)
}

func featureClear(ctx *flags.Context) error {
	pos, opt := flags.Flags()

	var seqinPath *string
	if isTerminal(os.Stdin.Fd()) {
		seqinPath = pos.String("input", "input sequence file (may be omitted if standard input is provided)")
	}

	seqoutPath := opt.String('o', "output", "-", "output sequence file (specifying `-` will force standard output)")
	format := opt.String('F', "format", "", "output file format (defaults to same as input)")

	if err := ctx.Parse(pos, opt); err != nil {
		return err
	}

	seqinFile := os.Stdin
	if seqinPath != nil && *seqinPath != "-" {
		f, err := os.Open(*seqinPath)
		if err != nil {
			return ctx.Raise(fmt.Errorf("failed to open file %q: %v", *seqinPath, err))
		}
		seqinFile = f
		defer seqinFile.Close()
	}

	seqoutFile := os.Stdout
	if *seqoutPath != "-" {
		f, err := os.Create(*seqoutPath)
		if err != nil {
			return ctx.Raise(fmt.Errorf("failed to create file %q: %v", *seqoutPath, err))
		}
		seqoutFile = f
		defer seqoutFile.Close()
	}

	filetype := seqio.Detect(*seqoutPath)
	if *format != "" {
		filetype = seqio.ToFileType(*format)
	}

	w := bufio.NewWriter(seqoutFile)

	scanner := seqio.NewAutoScanner(seqinFile)
	for scanner.Scan() {
		seq := scanner.Value()
		ff := seq.Features().Filter(gts.Key("source"))
		seq = gts.WithFeatures(seq, ff)
		formatter := seqio.NewFormatter(seq, filetype)
		_, err := formatter.WriteTo(w)
		if err != nil {
			return ctx.Raise(err)
		}
	}

	w.Flush()

	if err := scanner.Err(); err != nil {
		return ctx.Raise(fmt.Errorf("encountered error in scanner: %v", err))
	}

	return nil
}

func featureSelect(ctx *flags.Context) error {
	pos, opt := flags.Flags()

	selector := pos.String("selector", "feature selector (syntax: feature_key[/qualifier1[=regexp1]][/qualifier2[]=regexp2]])")

	var seqinPath *string
	if isTerminal(os.Stdin.Fd()) {
		seqinPath = pos.String("input", "input sequence file (may be omitted if standard input is provided)")
	}

	seqoutPath := opt.String('o', "output", "-", "output sequence file (specifying `-` will force standard output)")
	invert := opt.Switch('v', "invert-match", "select features that do not match the given criteria")
	format := opt.String('F', "format", "", "output file format (defaults to same as input)")

	if err := ctx.Parse(pos, opt); err != nil {
		return err
	}

	filter, err := gts.Selector(*selector)
	if err != nil {
		return ctx.Raise(fmt.Errorf("invalid selector syntax: %v", err))
	}
	if *invert {
		filter = gts.Not(filter)
	}
	filter = gts.Or(gts.Key("source"), filter)

	seqinFile := os.Stdin
	if seqinPath != nil && *seqinPath != "-" {
		f, err := os.Open(*seqinPath)
		if err != nil {
			return ctx.Raise(fmt.Errorf("failed to open file %q: %v", *seqinPath, err))
		}
		seqinFile = f
		defer seqinFile.Close()
	}

	seqoutFile := os.Stdout
	if *seqoutPath != "-" {
		f, err := os.Create(*seqoutPath)
		if err != nil {
			return ctx.Raise(fmt.Errorf("failed to create file %q: %v", *seqoutPath, err))
		}
		seqoutFile = f
		defer seqoutFile.Close()
	}

	filetype := seqio.Detect(*seqoutPath)
	if *format != "" {
		filetype = seqio.ToFileType(*format)
	}

	w := bufio.NewWriter(seqoutFile)

	scanner := seqio.NewAutoScanner(seqinFile)
	for scanner.Scan() {
		seq := scanner.Value()
		ff := seq.Features().Filter(filter)
		seq = gts.WithFeatures(seq, ff)
		formatter := seqio.NewFormatter(seq, filetype)
		_, err := formatter.WriteTo(w)
		if err != nil {
			return ctx.Raise(err)
		}
	}

	w.Flush()

	if err := scanner.Err(); err != nil {
		return ctx.Raise(fmt.Errorf("encountered error in scanner: %v", err))
	}

	return nil
}

func featureAnnotate(ctx *flags.Context) error {
	pos, opt := flags.Flags()

	featinPath := pos.String("feature_table", "feature table file containing features to merge")

	var seqinPath *string
	if isTerminal(os.Stdin.Fd()) {
		seqinPath = pos.String("input", "input sequence file (may be omitted if standard input is provided)")
	}

	seqoutPath := opt.String('o', "output", "-", "output sequence file (specifying `-` will force standard output)")
	format := opt.String('F', "format", "", "output file format (defaults to same as input)")

	if err := ctx.Parse(pos, opt); err != nil {
		return err
	}

	seqinFile := os.Stdin
	if seqinPath != nil && *seqinPath != "-" {
		f, err := os.Open(*seqinPath)
		if err != nil {
			return ctx.Raise(fmt.Errorf("failed to open file %q: %v", *seqinPath, err))
		}
		seqinFile = f
		defer seqinFile.Close()
	}

	featinFile, err := os.Open(*featinPath)
	if err != nil {
		return ctx.Raise(fmt.Errorf("failed to open file %q: %v", *featinPath, err))
	}

	seqoutFile := os.Stdout
	if *seqoutPath != "-" {
		f, err := os.Create(*seqoutPath)
		if err != nil {
			return ctx.Raise(fmt.Errorf("failed to create file %q: %v", *seqoutPath, err))
		}
		seqoutFile = f
		defer seqoutFile.Close()
	}

	filetype := seqio.Detect(*seqoutPath)
	if *format != "" {
		filetype = seqio.ToFileType(*format)
	}

	state := pars.NewState(featinFile)
	result, err := gts.FeatureTableParser("").Parse(state)
	featin := result.Value.(gts.FeatureTable)

	w := bufio.NewWriter(seqoutFile)

	scanner := seqio.NewAutoScanner(seqinFile)
	for scanner.Scan() {
		seq := scanner.Value()
		ff := seq.Features()
		for _, f := range featin {
			ff = ff.Insert(f)
		}
		seq = gts.WithFeatures(seq, ff)
		formatter := seqio.NewFormatter(seq, filetype)
		_, err := formatter.WriteTo(w)
		if err != nil {
			return ctx.Raise(err)
		}
	}

	w.Flush()

	if err := scanner.Err(); err != nil {
		return ctx.Raise(fmt.Errorf("encountered error in scanner: %v", err))
	}

	return nil
}

func featureExtract(ctx *flags.Context) error {
	pos, opt := flags.Flags()

	var seqinPath *string
	if isTerminal(os.Stdin.Fd()) {
		seqinPath = pos.String("input", "input sequence file (may be omitted if standard input is provided)")
	}

	outPath := opt.String('o', "output", "-", "output table file (specifying `-` will force standard output)")
	names := opt.StringSlice('n', "name", nil, "qualifier name(s) to select")
	delim := opt.String('d', "delimiter", "\t", "string to insert between columns")
	sep := opt.String('t', "separator", ",", "string to insert between qualifier values")
	nokey := opt.Switch(0, "no-key", "do not extract the feature key")
	noloc := opt.Switch(0, "no-location", "do not extract the feature location")
	empty := opt.Switch(0, "empty", "allow missing qualifiers to be extracted")

	if err := ctx.Parse(pos, opt); err != nil {
		return err
	}

	seqinFile := os.Stdin
	if seqinPath != nil && *seqinPath != "-" {
		f, err := os.Open(*seqinPath)
		if err != nil {
			return ctx.Raise(fmt.Errorf("failed to open file %q: %v", *seqinPath, err))
		}
		seqinFile = f
		defer seqinFile.Close()
	}

	outFile := os.Stdout
	if *outPath != "-" {
		f, err := os.Create(*outPath)
		if err != nil {
			return ctx.Raise(fmt.Errorf("failed to create file %q: %v", *outPath, err))
		}
		outFile = f
		defer outFile.Close()
	}

	w := bufio.NewWriter(outFile)

	fields := []string{}
	if !*nokey {
		fields = append(fields, "feature")
	}
	if !*noloc {
		fields = append(fields, "location")
	}
	fields = append(fields, *names...)
	header := fmt.Sprintf("%s\n", strings.Join(fields, *delim))
	_, err := io.WriteString(w, header)
	if err != nil {
		return ctx.Raise(err)
	}

	scanner := seqio.NewAutoScanner(seqinFile)
	for scanner.Scan() {
		seq := scanner.Value()

		for _, f := range seq.Features() {
			values := make([]string, 0)
			if !*nokey {
				values = append(values, f.Key)
			}
			if !*noloc {
				values = append(values, f.Location.String())
			}
			ok := true
			for _, name := range *names {
				value := f.Qualifiers.Get(name)
				if len(value) == 0 && !*empty {
					ok = false
				}
				values = append(values, strings.Join(value, *sep))
			}
			if ok {
				line := fmt.Sprintf("%s\n", strings.Join(values, *delim))
				_, err := io.WriteString(w, line)
				if err != nil {
					return ctx.Raise(err)
				}
			}
		}
	}

	w.Flush()

	if err := scanner.Err(); err != nil {
		return ctx.Raise(fmt.Errorf("encountered error in scanner: %v", err))
	}

	return nil
}

func featureSeq(ctx *flags.Context) error {
	pos, opt := flags.Flags()

	var seqinPath *string
	if isTerminal(os.Stdin.Fd()) {
		seqinPath = pos.String("input", "input sequence file (may be omitted if standard input is provided)")
	}

	seqoutPath := opt.String('o', "output", "-", "output sequence file (specifying `-` will force standard output)")
	format := opt.String('F', "format", "", "output file format (defaults to same as input)")

	if err := ctx.Parse(pos, opt); err != nil {
		return err
	}

	seqinFile := os.Stdin
	if seqinPath != nil && *seqinPath != "-" {
		f, err := os.Open(*seqinPath)
		if err != nil {
			return ctx.Raise(fmt.Errorf("failed to open file %q: %v", *seqinPath, err))
		}
		seqinFile = f
		defer seqinFile.Close()
	}

	seqoutFile := os.Stdout
	if *seqoutPath != "-" {
		f, err := os.Create(*seqoutPath)
		if err != nil {
			return ctx.Raise(fmt.Errorf("failed to create file %q: %v", *seqoutPath, err))
		}
		seqoutFile = f
		defer seqoutFile.Close()
	}

	filetype := seqio.Detect(*seqoutPath)
	if *format != "" {
		filetype = seqio.ToFileType(*format)
	}

	w := bufio.NewWriter(seqoutFile)

	scanner := seqio.NewAutoScanner(seqinFile)
	for scanner.Scan() {
		seq := scanner.Value()
		ff := seq.Features().Filter(gts.Not(gts.Key("source")))
		for _, f := range ff {
			if loc, ok := f.Location.(gts.Locatable); ok {
				out := loc.Locate(seq)
				formatter := seqio.NewFormatter(out, filetype)
				_, err := formatter.WriteTo(w)
				if err != nil {
					return ctx.Raise(err)
				}
			}
		}
	}

	w.Flush()

	if err := scanner.Err(); err != nil {
		return ctx.Raise(fmt.Errorf("encountered error in scanner: %v", err))
	}

	return nil
}