package gts

import (
	"io"
	"sort"
	"strings"

	ascii "gopkg.in/ktnyt/ascii.v1"
	pars "gopkg.in/ktnyt/pars.v2"
)

// Feature represents a single feature within a feature table.
type Feature struct {
	Key        string
	Location   Location
	Qualifiers Values

	order map[string]int
	proxy SequenceProxy
}

// NewFeature creates a new feature.
func NewFeature(key string, loc Location, qfs Values) Feature {
	return Feature{
		Key:        key,
		Location:   loc,
		Qualifiers: qfs,
	}
}

// Bytes satisfies the gts.Sequence interface.
func (f Feature) Bytes() []byte { return f.Location.Locate(f.proxy).Bytes() }

// Insert a sequence at the specified position.
func (f Feature) Insert(pos int, seq Sequence) { f.Insert(pos, seq) }

// Delete given number of bases from the specified position.
func (f Feature) Delete(pos, cnt int) { f.Delete(pos, cnt) }

// Replace the bases from the specified position with the given sequence.
func (f Feature) Replace(pos int, seq Sequence) { f.Replace(pos, seq) }

// Translation returns the translation of the feature if available. it will
// return nil otherwise.
func (f Feature) Translation() Sequence {
	if values := f.Qualifiers.Get("translation"); len(values) != 0 {
		s := values[0]
		return Seq(strings.ReplaceAll(s, "\n", ""))
	}
	return nil
}

// Format creates a FeatureFormatter object for the qualifier with the given
// prefix and depth. If the Feature object was created by parsing some input,
// the qualifier values will be in the same order as in the input source. The
// exception to this rule is the `translation` qualifier which will always be
// written last. Qualifiers given during runtime will be sorted in ascending
// alphabetical order and written after the qualifiers present in the source.
func (f Feature) Format(prefix string, depth int) FeatureFormatter {
	return FeatureFormatter{f, prefix, depth}
}

// FeatureFormatter formats a Feature object with the given prefix and depth.
type FeatureFormatter struct {
	Feature Feature
	Prefix  string
	Depth   int
}

// String satisfies the fmt.Stringer interface.
func (ff FeatureFormatter) String() string {
	builder := strings.Builder{}
	builder.WriteString(ff.Prefix)
	builder.WriteString(ff.Feature.Key)

	padding := strings.Repeat(" ", ff.Depth-builder.Len())
	prefix := ff.Prefix + strings.Repeat(" ", ff.Depth-len(ff.Prefix))

	builder.WriteString(padding)
	builder.WriteString(ff.Feature.Location.String())

	ordered := make([]string, len(ff.Feature.order))
	remains := []string{}

	hasTranslate := false

	for name := range ff.Feature.Qualifiers {
		index, ok := ff.Feature.order[name]
		switch {
		case ok:
			ordered[index] = name
		case name == "translation":
			hasTranslate = true
		default:
			remains = append(remains, name)
		}
	}

	for i, name := range ordered {
		if name == "" {
			ordered = append(ordered[:i], ordered[i+1:]...)
		}
	}

	sort.Strings(remains)

	names := append(ordered, remains...)

	if hasTranslate {
		names = append(names, "translation")
	}

	for _, name := range names {
		for _, value := range ff.Feature.Qualifiers[name] {
			q := Qualifier{name, value}
			builder.WriteByte('\n')
			builder.WriteString(q.Format(prefix).String())
		}
	}

	return builder.String()
}

// WriteTo satisfies the io.WriteTo interface.
func (ff FeatureFormatter) WriteTo(w io.Writer) (int, error) {
	return w.Write([]byte(ff.String()))
}

// ByLocation implements sort.Interface for []Feature by location.
type ByLocation []Feature

// Len is the number of elements in the feature table.
func (ff ByLocation) Len() int { return len(ff) }

// Less reports whether the element with index i should sort before the element
// with index j.
func (ff ByLocation) Less(i, j int) bool {
	a, b := ff[i], ff[j]
	if a.Key == "source" && b.Key != "source" {
		return true
	}
	if b.Key == "source" && a.Key != "source" {
		return false
	}
	return LocationLess(ff[i].Location, ff[j].Location)
}

// Swap the elements with indices i and j.
func (ff ByLocation) Swap(i, j int) {
	ff[i], ff[j] = ff[j], ff[i]
}

// FeatureList represents an INSDC feature table. The features are sorted by
// Location in ascending order.
type FeatureList []Feature

// Format creates a FeatureFormatter object for the qualifier with the given
// prefix and depth.
func (ff FeatureList) Format(prefix string, depth int) FeatureListFormatter {
	return FeatureListFormatter{ff, prefix, depth}
}

// Select the features in the list matching the selector criteria.
func (ff FeatureList) Select(sel FeatureSelector) []Feature {
	idx, n := make([]int, len(ff)), 0
	for i, f := range ff {
		if sel(f) {
			idx[n] = i
			n++
		}
	}

	selected := make([]Feature, n)
	for i, j := range idx[:n] {
		selected[i] = ff[j]
	}
	return selected
}

// Insert the feature to the feature table at the given position. Note that
// inserting a feature that disrupts the sortedness of the features will
// inevitably lead to predictable yet unconventional behavior when the Add
// method is called later. Use Add instead if this is not desired.
func (ff *FeatureList) Insert(i int, f Feature) {
	features := append(*ff, Feature{})
	copy(features[i+1:], features[i:])
	features[i] = f
	*ff = features
}

// Add the feature to the feature table. The feature will be inserted in the
// sorted position with the exception of sources.
func (ff *FeatureList) Add(f Feature) {
	n := 0
	for n < len(*ff) && (*ff)[n].Key == "source" {
		n++
	}

	switch f.Key {
	case "source":
		ff.Insert(n, f)
	default:
		i := sort.Search(len((*ff)[n:]), func(i int) bool {
			return LocationLess(f.Location, (*ff)[n+i].Location)
		})
		ff.Insert(n+i, f)
	}
}

// FeatureListFormatter formats a FeatureList object with the given prefix and
// depth.
type FeatureListFormatter struct {
	FeatureList FeatureList
	Prefix      string
	Depth       int
}

// String satisfies the fmt.Stringer interface.
func (ff FeatureListFormatter) String() string {
	b := strings.Builder{}
	for i, f := range ff.FeatureList {
		if i != 0 {
			b.WriteByte('\n')
		}
		f.Format(ff.Prefix, ff.Depth).WriteTo(&b)
	}
	return b.String()
}

// WriteTo satisfies the io.WriterTo interface.
func (ff FeatureListFormatter) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte(ff.String()))
	return int64(n), err
}

type keyline struct {
	pre int
	key string
	pst int
	loc Location
}

func featureKeylineParser(prefix string, depth int) pars.Parser {
	parser := pars.Seq(prefix, pars.Word(ascii.IsSnake)).Child(1)
	return func(state *pars.State, result *pars.Result) error {
		if err := parser(state, result); err != nil {
			return err
		}
		key := string(result.Token)
		remain := pars.Seq(
			pars.Count(byte(' '), depth-len(prefix+key)),
			LocationParser, pars.EOL,
		).Child(1)
		if err := remain(state, result); err != nil {
			return err
		}
		loc := result.Value.(Location)
		result.SetValue(keyline{0, key, 0, loc})
		return nil
	}
}

// FeatureListParser attempts to match an INSDC feature table.
func FeatureListParser(prefix string) pars.Parser {
	firstParser := pars.Seq(
		prefix, pars.Spaces,
		pars.Word(ascii.IsSnake), pars.Spaces,
		LocationParser, pars.EOL,
	).Map(func(result *pars.Result) error {
		children := result.Children
		pre := len(children[1].Token)
		key := string(children[2].Token)
		pst := len(children[3].Token)
		loc := children[4].Value.(Location)
		result.SetValue(keyline{pre, key, pst, loc})
		return nil
	})

	return func(state *pars.State, result *pars.Result) error {
		if err := firstParser(state, result); err != nil {
			return err
		}
		tmp := result.Value.(keyline)
		pre, key, pst, loc := tmp.pre, tmp.key, tmp.pst, tmp.loc
		depth := pre + len(key) + pst

		keylineParser := featureKeylineParser(prefix+strings.Repeat(" ", pre), depth)

		qualifierParser := QualifierParser(prefix + strings.Repeat(" ", depth))
		qualifiersParser := pars.Many(pars.Seq(qualifierParser, pars.EOL).Child(0))

		// Does not return error by definition.
		qualifiersParser(state, result)

		qfs := Values{}
		order := make(map[string]int)

		for _, child := range result.Children {
			q := child.Value.(Qualifier)
			qfs.Add(q.Name, q.Value)
			if _, ok := order[q.Name]; q.Name != "translation" && !ok {
				order[q.Name] = len(order)
			}
		}

		ff := []Feature{{
			Key:        key,
			Location:   loc,
			Qualifiers: qfs,
			order:      order,
		}}

		for keylineParser(state, result) == nil {
			tmp := result.Value.(keyline)
			key, loc := tmp.key, tmp.loc

			// Does not return error by definition.
			qualifiersParser(state, result)

			qfs := Values{}
			order := make(map[string]int)

			for _, child := range result.Children {
				q := child.Value.(Qualifier)
				qfs.Add(q.Name, q.Value)
				if _, ok := order[q.Name]; q.Name != "translation" && !ok {
					order[q.Name] = len(order)
				}
			}

			ff = append(ff, Feature{
				Key:        key,
				Location:   loc,
				Qualifiers: qfs,
				order:      order,
			})
		}

		result.SetValue(ff)
		return nil
	}
}
