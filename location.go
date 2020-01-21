package gts

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	ascii "gopkg.in/ascii.v1"
	pars "gopkg.in/pars.v2"
)

// Location represents a feature location as defined by the INSDC.
type Location interface {
	// Locate the sequence at the pointing location.
	Locate(seq Sequence) Sequence

	// Len returns the length spanned by the location.
	Len() int

	// String satisfies the fmt.Stringer interface.
	String() string

	// Shift the location by the given amount if needed.
	// Returns false if the shift invalidates the location.
	Shift(offset, amount int) bool

	// Map the given local index to a global index.
	Map(index int) int
}

// LocationLess tests if the one location is smaller than the other.
func LocationLess(a, b Location) bool {
	if a.Map(0) < b.Map(0) {
		return true
	}
	if b.Map(0) < a.Map(0) {
		return false
	}
	if a.Map(a.Len()-1) < b.Map(b.Len()-1) {
		return true
	}
	if b.Map(b.Len()-1) < a.Map(a.Len()-1) {
		return false
	}
	return false
}

// PointLocation represents a single point.
type PointLocation struct{ Position int }

// NewPointLocation creates a new PointLocation.
func NewPointLocation(pos int) *PointLocation {
	return &PointLocation{Position: pos}
}

// Locate the sequence at the pointing location.
func (loc PointLocation) Locate(seq Sequence) Sequence {
	return Slice(seq, loc.Position, loc.Position+1)
}

// Len returns the length spanned by the location.
func (loc PointLocation) Len() int {
	return 1
}

// String satisfies the fmt.Stringer interface.
func (loc PointLocation) String() string {
	return strconv.Itoa(loc.Position + 1)
}

// Shift the location position[s] if needed.
// Returns false if the shift invalidates the location.
func (loc *PointLocation) Shift(offset, amount int) bool {
	if amount == 0 || loc.Position < offset {
		return true
	}
	if amount < 0 && loc.Position < offset-amount {
		return false
	}
	loc.Position += amount
	return true
}

// Map the given local index to a global index.
func (loc PointLocation) Map(index int) int {
	if index < 0 {
		panic(fmt.Errorf("invalid `%T` index %d (index must be non-negative)", loc, index))
	}
	if index >= loc.Len() {
		panic(fmt.Errorf("index [%d] is outside of `%T` with length %d", index, loc, loc.Len()))
	}
	return loc.Position
}

func shiftRange(a, b, i, n int) (int, int, bool) {
	switch {
	case n > 0:
		if i <= a {
			a += n
		}
		if i <= b {
			b += n
		}
		return a, b, true
	case n < 0:
		c, d := a, b
		if i-n <= c {
			c += n
		}
		if i-n <= d {
			d += n
		}
		if c < d-1 {
			return c, d, true
		}
		return a, b, false
	default:
		return a, b, true
	}
}

// RangeLocation represents a range of locations.
type RangeLocation struct {
	Start    int
	End      int
	Partial5 bool
	Partial3 bool
}

// NewRangeLocation creates a new RangeLocation.
func NewRangeLocation(start, end int) *RangeLocation {
	return NewPartialRangeLocation(start, end, false, false)
}

// NewPartialRangeLocation creates a new partial RangeLocation.
func NewPartialRangeLocation(start, end int, p5, p3 bool) *RangeLocation {
	return &RangeLocation{
		Start:    start,
		End:      end,
		Partial5: p5,
		Partial3: p3,
	}
}

// Locate the sequence at the pointing location.
func (loc RangeLocation) Locate(seq Sequence) Sequence {
	return Slice(seq, loc.Start, loc.End)
}

// Len returns the length spanned by the location.
func (loc RangeLocation) Len() int {
	return loc.End - loc.Start
}

// String satisfies the fmt.Stringer interface.
func (loc RangeLocation) String() string {
	p5, p3 := "", ""
	if loc.Partial5 {
		p5 = "<"
	}
	if loc.Partial3 {
		p3 = ">"
	}
	return fmt.Sprintf("%s%d..%s%d", p5, loc.Start+1, p3, loc.End)
}

// Shift the location position[s] if needed.
// Returns false if the shift invalidates the location.
func (loc *RangeLocation) Shift(offset, amount int) (ok bool) {
	loc.Start, loc.End, ok = shiftRange(loc.Start, loc.End, offset, amount)
	return
}

// Map the given local index to a global index.
func (loc RangeLocation) Map(index int) int {
	if index < 0 {
		panic(fmt.Errorf("invalid `%T` index %d (index must be non-negative)", loc, index))
	}
	if index >= loc.Len() {
		panic(fmt.Errorf("index [%d] is outside of `%T` with length %d", index, loc, loc.Len()))
	}
	return loc.Start + index
}

// AmbiguousLocation represents an ambiguous location.
type AmbiguousLocation struct {
	Start int
	End   int
}

// NewAmbiguousLocation creates a new ambiguous location.
func NewAmbiguousLocation(start, end int) *AmbiguousLocation {
	return &AmbiguousLocation{Start: start, End: end}
}

// Locate the sequence at the pointing location.
func (loc AmbiguousLocation) Locate(seq Sequence) Sequence {
	return Slice(seq, loc.Start, loc.End)
}

// Len returns the length spanned by the location.
func (loc AmbiguousLocation) Len() int {
	return loc.End - loc.Start
}

// String satisfies the fmt.Stringer interface.
func (loc AmbiguousLocation) String() string {
	return fmt.Sprintf("%d.%d", loc.Start+1, loc.End)
}

// Shift the location position[s] if needed.
// Returns false if the shift invalidates the location.
func (loc *AmbiguousLocation) Shift(offset, amount int) (ok bool) {
	loc.Start, loc.End, ok = shiftRange(loc.Start, loc.End, offset, amount)
	return
}

// Map the given local index to a global index.
func (loc AmbiguousLocation) Map(index int) int {
	if index < 0 {
		panic(fmt.Errorf("invalid `%T` index %d (index must be non-negative)", loc, index))
	}
	if index >= loc.Len() {
		panic(fmt.Errorf("index [%d] is outside of `%T` with length %d", index, loc, loc.Len()))
	}
	return loc.Start + index
}

// BetweenLocation represents a location between two points.
type BetweenLocation struct {
	Start int
	End   int
}

// NewBetweenLocation creates a new BetweenLocation.
func NewBetweenLocation(start, end int) *BetweenLocation {
	return &BetweenLocation{Start: start, End: end}
}

// Locate the sequence at the pointing location.
func (loc BetweenLocation) Locate(seq Sequence) Sequence {
	return Slice(seq, loc.Start, loc.End)
}

// Len returns the length spanned by the location.
func (loc BetweenLocation) Len() int {
	return loc.End - loc.Start
}

// String satisfies the fmt.Stringer interface.
func (loc BetweenLocation) String() string {
	return fmt.Sprintf("%d^%d", loc.Start+1, loc.End)
}

// Shift the location position[s] if needed.
// Returns false if the shift invalidates the location.
func (loc *BetweenLocation) Shift(offset, amount int) (ok bool) {
	loc.Start, loc.End, ok = shiftRange(loc.Start, loc.End, offset, amount)
	return
}

// Map the given local index to a global index.
func (loc BetweenLocation) Map(index int) int {
	if index < 0 {
		panic(fmt.Errorf("invalid `%T` index %d (index must be non-negative)", loc, index))
	}
	if index >= loc.Len() {
		panic(fmt.Errorf("index [%d] is outside of `%T` with length %d", index, loc, loc.Len()))
	}
	return loc.Start + index
}

// ComplementLocation represents the complement region of a location.
type ComplementLocation struct {
	Location Location
}

// NewComplementLocation creates a new ComplementLocation.
func NewComplementLocation(loc Location) *ComplementLocation {
	return &ComplementLocation{Location: loc}
}

// Locate the sequence at the pointing location.
func (loc ComplementLocation) Locate(seq Sequence) Sequence {
	return Complement(loc.Location.Locate(seq))
}

// Len returns the length spanned by the location.
func (loc ComplementLocation) Len() int {
	return loc.Location.Len()
}

// String satisfies the fmt.Stringer interface.
func (loc ComplementLocation) String() string {
	return fmt.Sprintf("complement(%s)", loc.Location.String())
}

// Shift the location position[s] if needed.
// Returns false if the shift invalidates the location.
func (loc *ComplementLocation) Shift(offset, amount int) bool {
	return loc.Location.Shift(offset, amount)
}

// Map the given local index to a global index.
func (loc ComplementLocation) Map(index int) int {
	return loc.Location.Map(index)
}

// JoinLocation represents multiple joined locations.
type JoinLocation struct {
	Locations []Location
}

// NewJoinLocation creates a new JoinLocation.
func NewJoinLocation(locs []Location) *JoinLocation {
	return &JoinLocation{Locations: locs}
}

// Locate the sequence at the pointing location.
func (loc JoinLocation) Locate(seq Sequence) Sequence {
	r := make([]byte, loc.Len())
	i := 0
	for _, l := range loc.Locations {
		copy(r[i:], l.Locate(seq).Bytes())
		i += l.Len()
	}
	return New(seq.Info(), r)
}

// Len returns the length spanned by the location.
func (loc JoinLocation) Len() int {
	length := 0
	for _, l := range loc.Locations {
		length += l.Len()
	}
	return length
}

// String satisfies the fmt.Stringer interface.
func (loc JoinLocation) String() string {
	tmp := make([]string, len(loc.Locations))
	for i := range loc.Locations {
		tmp[i] = loc.Locations[i].String()
	}
	return fmt.Sprintf("join(%s)", strings.Join(tmp, ","))
}

// Shift the location position[s] if needed.
// Returns false if the shift invalidates the location.
func (loc *JoinLocation) Shift(pos, n int) bool {
	ok := true
	for i := range loc.Locations {
		if !loc.Locations[i].Shift(pos, n) {
			ok = false
		}
	}
	return ok
}

// Map the given local index to a global index.
func (loc JoinLocation) Map(index int) int {
	if index < 0 {
		panic(fmt.Errorf("invalid `%T` index %d (index must be non-negative)", loc, index))
	}
	for _, l := range loc.Locations {
		if index < l.Len() {
			return l.Map(index)
		}
		index -= l.Len()
	}
	panic(fmt.Errorf("index [%d] is outside of `%T` with length %d", index, loc, loc.Len()))
}

// OrderLocation represents a group of locations.
type OrderLocation struct {
	Locations []Location
}

// NewOrderLocation creates a new OrderLocation.
func NewOrderLocation(locs []Location) *OrderLocation {
	return &OrderLocation{Locations: locs}
}

// Locate the sequence at the pointing location.
func (loc OrderLocation) Locate(seq Sequence) Sequence {
	r := make([]byte, loc.Len())
	i := 0
	for _, l := range loc.Locations {
		copy(r[i:], l.Locate(seq).Bytes())
		i += l.Len()
	}
	return New(seq.Info(), r)
}

// Len returns the length spanned by the location.
func (loc OrderLocation) Len() int {
	length := 0
	for _, l := range loc.Locations {
		length += l.Len()
	}
	return length
}

// String satisfies the fmt.Stringer interface.
func (loc OrderLocation) String() string {
	tmp := make([]string, len(loc.Locations))
	for i := range loc.Locations {
		tmp[i] = loc.Locations[i].String()
	}
	return fmt.Sprintf("order(%s)", strings.Join(tmp, ","))
}

// Shift the location position[s] if needed.
// Returns false if the shift invalidates the location.
func (loc *OrderLocation) Shift(pos, n int) bool {
	ok := true
	for i := range loc.Locations {
		if !loc.Locations[i].Shift(pos, n) {
			ok = false
		}
	}
	return ok
}

// Map the given local index to a global index.
func (loc OrderLocation) Map(index int) int {
	if index < 0 {
		panic(fmt.Errorf("invalid `%T` index %d (index must be non-negative)", loc, index))
	}
	for _, l := range loc.Locations {
		if index < l.Len() {
			return l.Map(index)
		}
		index -= l.Len()
	}
	panic(fmt.Errorf("index [%d] is outside of `%T` with length %d", index, loc, loc.Len()))
}

// LocationParser attempts to parse some location.
var LocationParser pars.Parser

// PointLocationParser attempts to parse a PointLocation.
func PointLocationParser(state *pars.State, result *pars.Result) error {
	if err := pars.Int(state, result); err != nil {
		return err
	}
	n := result.Value.(int) - 1
	result.SetValue(NewPointLocation(n))
	return nil
}

var pointLocationParser = pars.Parser(pars.Int).Map(func(result *pars.Result) error {
	n := result.Value.(int)
	loc := NewPointLocation(n - 1)
	result.SetValue(loc)
	return nil
})

// RangeLocationParser attempts to parse a RangeLocation.
func RangeLocationParser(state *pars.State, result *pars.Result) error {
	state.Push()
	c, err := pars.Next(state)
	if err != nil {
		state.Pop()
		return err
	}
	partial5 := false
	if c == '<' {
		partial5 = true
		state.Advance()
	}
	if err := pars.Int(state, result); err != nil {
		state.Pop()
		return err
	}
	start := result.Value.(int) - 1
	if err := state.Request(2); err != nil {
		state.Pop()
		return err
	}
	if !bytes.Equal(state.Buffer(), []byte("..")) {
		state.Pop()
		return pars.NewError("expected `..`", state.Position())
	}
	state.Advance()
	c, err = pars.Next(state)
	partial3 := false
	if c == '>' {
		partial3 = true
		state.Advance()
	}
	if err := pars.Int(state, result); err != nil {
		state.Pop()
		return err
	}
	end := result.Value.(int)
	c, err = pars.Next(state)
	if err != nil && c == '>' {
		partial3 = true
		state.Advance()
	}
	result.SetValue(NewPartialRangeLocation(start, end, partial5, partial3))
	state.Drop()
	return nil
}

// AmbiguousLocationParser attempts to parse a AmbiguousLocation.
func AmbiguousLocationParser(state *pars.State, result *pars.Result) error {
	state.Push()
	if err := pars.Int(state, result); err != nil {
		state.Pop()
		return err
	}
	start := result.Value.(int) - 1
	c, err := pars.Next(state)
	if err != nil {
		state.Pop()
		return err
	}
	if c != '.' {
		state.Pop()
		return pars.NewError("expected `.`", state.Position())
	}
	state.Advance()
	if err := pars.Int(state, result); err != nil {
		state.Pop()
		return err
	}
	end := result.Value.(int)
	result.SetValue(NewAmbiguousLocation(start, end))
	state.Drop()
	return nil
}

// BetweenLocationParser attempts to parse a BetweenLocation.
func BetweenLocationParser(state *pars.State, result *pars.Result) error {
	state.Push()
	if err := pars.Int(state, result); err != nil {
		state.Pop()
		return err
	}
	start := result.Value.(int) - 1
	c, err := pars.Next(state)
	if err != nil {
		state.Pop()
		return err
	}
	if c != '^' {
		state.Pop()
		return pars.NewError("expected `.`", state.Position())
	}
	state.Advance()
	if err := pars.Int(state, result); err != nil {
		state.Pop()
		return err
	}
	end := result.Value.(int)
	result.SetValue(NewBetweenLocation(start, end))
	state.Drop()
	return nil
}

// ComplementLocationParser attempts to parse a ComplementLocation.
func ComplementLocationParser(state *pars.State, result *pars.Result) error {
	state.Push()
	if err := state.Request(11); err != nil {
		state.Pop()
		return err
	}
	if !bytes.Equal(state.Buffer(), []byte("complement(")) {
		state.Pop()
		return pars.NewError("expected `complement(`", state.Position())
	}
	state.Advance()
	if err := LocationParser(state, result); err != nil {
		state.Pop()
		return err
	}
	c, err := pars.Next(state)
	if err != nil {
		state.Pop()
		return err
	}
	if c != ')' {
		state.Pop()
		return pars.NewError("expected `)`", state.Position())
	}
	state.Advance()
	result.SetValue(NewComplementLocation(result.Value.(Location)))
	state.Drop()
	return nil
}

func locationDelimiter(state *pars.State, result *pars.Result) bool {
	state.Push()
	c, err := pars.Next(state)
	if err != nil {
		state.Pop()
		return false
	}
	if c != ',' {
		state.Pop()
		return false
	}
	state.Advance()
	c, err = pars.Next(state)
	for ascii.IsSpace(c) && err == nil {
		state.Advance()
		c, err = pars.Next(state)
	}
	state.Drop()
	return true
}

func multipleLocationParser(state *pars.State, result *pars.Result) error {
	state.Push()
	if err := LocationParser(state, result); err != nil {
		state.Pop()
		return err
	}
	locs := []Location{result.Value.(Location)}
	for locationDelimiter(state, result) {
		if err := LocationParser(state, result); err != nil {
			state.Pop()
			return err
		}
		locs = append(locs, result.Value.(Location))
	}
	result.SetValue(locs)
	state.Drop()
	return nil
}

// JoinLocationParser attempts to parse a JoinLocation.
func JoinLocationParser(state *pars.State, result *pars.Result) error {
	state.Push()
	if err := state.Request(5); err != nil {
		state.Pop()
		return err
	}
	if !bytes.Equal(state.Buffer(), []byte("join(")) {
		state.Pop()
		return pars.NewError("expected `join(`", state.Position())
	}
	state.Advance()
	if err := multipleLocationParser(state, result); err != nil {
		return err
	}
	c, err := pars.Next(state)
	if err != nil {
		state.Pop()
		return err
	}
	if c != ')' {
		state.Pop()
		return pars.NewError("expected `)`", state.Position())
	}
	state.Advance()
	result.SetValue(NewJoinLocation(result.Value.([]Location)))
	state.Drop()
	return nil
}

// OrderLocationParser attempts to parse a OrderLocation.
func OrderLocationParser(state *pars.State, result *pars.Result) error {
	state.Push()
	if err := state.Request(6); err != nil {
		state.Pop()
		return err
	}
	if !bytes.Equal(state.Buffer(), []byte("order(")) {
		state.Pop()
		return pars.NewError("expected `order(`", state.Position())
	}
	state.Advance()
	if err := multipleLocationParser(state, result); err != nil {
		return err
	}
	c, err := pars.Next(state)
	if err != nil {
		state.Pop()
		return err
	}
	if c != ')' {
		state.Pop()
		return pars.NewError("expected `)`", state.Position())
	}
	state.Advance()
	result.SetValue(NewOrderLocation(result.Value.([]Location)))
	state.Drop()
	return nil
}

var errNotLocation = errors.New("string is not a Location")

// AsLocation attempts to interpret the given string as a Location.
func AsLocation(s string) (Location, error) {
	state := pars.FromString(s)
	result := pars.Result{}
	parser := pars.Exact(LocationParser).Error(errNotLocation)
	if err := parser(state, &result); err != nil {
		return nil, err
	}
	return result.Value.(Location), nil
}

func init() {
	LocationParser = pars.Any(
		RangeLocationParser,
		OrderLocationParser,
		JoinLocationParser,
		ComplementLocationParser,
		AmbiguousLocationParser,
		BetweenLocationParser,
		PointLocationParser,
	)
}
