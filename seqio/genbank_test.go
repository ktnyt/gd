package gts

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-ascii/ascii"
	"github.com/go-gts/gts"
	"github.com/go-gts/gts/testutils"
	"github.com/go-pars/pars"
)

func TestGenBankIO(t *testing.T) {
	in := testutils.ReadGolden(t)
	state := pars.FromString(in)
	parser := pars.AsParser(GenBankParser)

	result, err := parser.Parse(state)
	if err != nil {
		t.Errorf("parser returned %v\nBuffer:\n%q", err, string(result.Token))
	}

	switch gb := result.Value.(type) {
	case GenBank:
		data := gb.Bytes()
		if len(data) != gts.Len(gb) {
			t.Errorf("len(data) = %d, want %d", len(data), gts.Len(gb))
			return
		}
		if gb.Info() == nil {
			t.Error("gb.Info() is nil")
			return
		}
		if gb.Features() == nil {
			t.Error("gb.Features() is nil")
			return
		}
		for i, c := range data {
			if !ascii.IsLetterFilter(c) {
				t.Errorf("origin contains `%U` at byte %d, expected a sequence character", c, i+1)
				return
			}
		}
		b := strings.Builder{}
		f := GenBankFormatter{&gb}
		n, err := f.WriteTo(&b)
		if int(n) != len([]byte(in)) || err != nil {
			t.Errorf("f.WriteTo(&b) = (%d, %v), want %d, nil", n, err, len(in))
			return
		}
		out := b.String()
		testutils.Diff(t, in, out)
	default:
		t.Errorf("result.Value.(type) = %T, want %T", gb, GenBank{})
	}
}

var genbankIOFailTests = []string{
	"NC_001422               5386 bp ss-DNA     circular PHG 06-JUL-2018",
	"" +
		"LOCUS       NC_001422               5386 bp ss-DNA     circular PHG 06-JUL-2018\n" +
		"foo",
	"" +
		"LOCUS       NC_001422               5386 bp ss-DNA     circular PHG 06-JUL-2018\n" +
		"DEFINITION",
	"" +
		"LOCUS       NC_001422               5386 bp ss-DNA     circular PHG 06-JUL-2018\n" +
		"DBLINK      FOO",
	"" +
		"LOCUS       NC_001422               5386 bp ss-DNA     circular PHG 06-JUL-2018\n" +
		"SOURCE      Escherichia virus phiX174\n" +
		"  ORGANISM Escherichia virus phiX174",
	"" +
		"LOCUS       NC_001422               5386 bp ss-DNA     circular PHG 06-JUL-2018\n" +
		"REFERENCE   1",
	"" +
		"LOCUS       NC_001422               5386 bp ss-DNA     circular PHG 06-JUL-2018\n" +
		"REFERENCE   1  (bases 2380 to 2512; 2593 to 2786; 2788 to 2947)\n" +
		"  AUTHORS  Air,G.M., Els,M.C., Brown,L.E., Laver,W.G. and Webster,R.G.",
	"" +
		"LOCUS       NC_001422               5386 bp ss-DNA     circular PHG 06-JUL-2018\n" +
		"FEATURES             Location/Qualifiers",
	"" +
		"LOCUS       TEST_DATA                 20 bp    DNA     linear   UNA 14-MAY-2020\n" +
		"ORIGIN      \n",
	"" +
		"LOCUS       TEST_DATA                 20 bp    DNA     linear   UNA 14-MAY-2020\n" +
		"ORIGIN      \n" +
		"       1 gagttttatc gcttccatga",
	"" +
		"LOCUS       TEST_DATA                 20 bp    DNA     linear   UNA 14-MAY-2020\n" +
		"ORIGIN      \n" +
		"        1 gagttttatcgcttccatga",
	"" +
		"LOCUS       TEST_DATA                 20 bp    DNA     linear   UNA 14-MAY-2020\n" +
		"ORIGIN      \n" +
		"        1  gagttttatc gcttccatga",
	"" +
		"LOCUS       NC_001422               5386 bp ss-DNA     circular PHG 06-JUL-2018\n" +
		"FOO         ",
}

func TestGenBankIOFail(t *testing.T) {
	parser := pars.AsParser(GenBankParser)
	for _, in := range genbankIOFailTests {
		state := pars.FromString(in)
		if _, err := parser.Parse(state); err == nil {
			t.Errorf("while parsing`\n%s\n`: expected error", in)
		}
	}

	w := bytes.Buffer{}
	n, err := GenBankFormatter{gts.New(nil, nil, nil)}.WriteTo(&w)
	if n != 0 || err == nil {
		t.Errorf("formatting an empty Sequence should return an error")
	}
}