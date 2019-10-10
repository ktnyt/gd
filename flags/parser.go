package flags

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type Command func(Values, *Parser, []string) error

type positional struct {
	Name  string
	Value Value
}

type Parser struct {
	program   string
	version   string
	values    map[string]Value
	switches  map[string]*BoolValue
	usages    map[string]string
	aliases   map[byte]string
	extras    []string
	commands  map[string]Command
	mandatory []positional
	optional  []positional
}

func NewParser(program, version string) *Parser {
	return &Parser{
		program:   program,
		version:   version,
		values:    make(map[string]Value),
		switches:  make(map[string]*BoolValue),
		usages:    make(map[string]string),
		aliases:   make(map[byte]string),
		extras:    make([]string, 0),
		commands:  make(map[string]Command),
		mandatory: make([]positional, 0),
		optional:  make([]positional, 0),
	}
}

func (parser *Parser) Command(name string, cmd Command) {
	parser.commands[name] = cmd
}

func (parser *Parser) Mandatory(name string) *string {
	p := NewStringValue("")
	parser.mandatory = append(parser.mandatory, positional{name, p})
	return (*string)(p)
}

func (parser *Parser) Optional(name string) *string {
	p := NewStringValue("")
	parser.optional = append(parser.optional, positional{name, p})
	return (*string)(p)
}

func (parser *Parser) Switch(short byte, long string, usage string) *bool {
	p := NewBoolValue(false)
	parser.switches[long] = p
	parser.usages[long] = usage
	if short != 0 {
		parser.aliases[short] = long
	}
	return (*bool)(p)
}

func (parser *Parser) Bool(short byte, long string, value bool, usage string) *bool {
	p := NewBoolValue(value)
	parser.Register(short, long, p, usage)
	return (*bool)(p)
}

func (parser *Parser) Int(short byte, long string, value int, usage string) *int {
	p := NewIntValue(value)
	parser.Register(short, long, p, usage)
	return (*int)(p)
}

func (parser *Parser) String(short byte, long string, value string, usage string) *string {
	p := NewStringValue(value)
	parser.Register(short, long, p, usage)
	return (*string)(p)
}

func (parser *Parser) Strings(short byte, long string, value []string, usage string) *[]string {
	if value == nil {
		value = make([]string, 0)
	}
	p := NewStringsValue(value)
	parser.Register(short, long, p, usage)
	return (*[]string)(p)
}

func (parser *Parser) Register(short byte, long string, value Value, usage string) {
	parser.values[long] = value
	parser.usages[long] = usage
	if short != 0 {
		parser.aliases[short] = long
	}
}

func (parser Parser) getLongName(short byte) (string, error) {
	if name, ok := parser.aliases[short]; ok {
		return name, nil
	}
	return "", fmt.Errorf("unexpected argument alias `%c`", short)
}

func (parser *Parser) trySetLong(name, value string) error {
	if p, ok := parser.values[name]; ok {
		return p.Set(value)
	}
	return fmt.Errorf("unexpected argument name `%s`", name)
}

func (parser *Parser) trySetBoolTrue(name string) error {
	if p, ok := parser.values[name]; ok {
		b, ok := p.(*BoolValue)
		if !ok {
			return fmt.Errorf("argument value expected for flag `%s`", name)
		}
		*b = BoolValue(true)
		return nil
	}

	if p, ok := parser.switches[name]; ok {
		*p = BoolValue(true)
		return nil
	}

	return fmt.Errorf("argument value expected for flag `%s`", name)
}

func (parser *Parser) handleLong(name string, args []string) ([]string, error) {
	if name == "help" {
		return nil, errors.New(parser.Help())
	}

	if name == "version" {
		return nil, errors.New(fmt.Sprintf("version: %s", parser.version))
	}

	if strings.Contains(name, "=") {
		split := strings.SplitN(name, "=", 2)
		return args, parser.trySetLong(split[0], split[1])
	}

	head, tail := args[0], args[1:]

	if strings.HasPrefix(head, "-") {
		return args, parser.trySetBoolTrue(name)
	}

	if parser.trySetLong(name, head) == nil {
		return tail, nil
	}

	if err := parser.trySetBoolTrue(name); err != nil {
		return nil, err
	}

	return args, nil
}

func (parser *Parser) handleShort(bytes []byte, args []string) ([]string, error) {
	for i, short := range bytes {
		if short == 'h' {
			return nil, errors.New(parser.Help())
		}

		name, err := parser.getLongName(short)
		if err != nil {
			return nil, err
		}

		if i+1 == len(bytes) && len(args) > 0 {
			head, tail := args[0], args[1:]
			if err := parser.trySetLong(name, head); err == nil {
				return tail, nil
			}
		}

		if err := parser.trySetBoolTrue(name); err != nil {
			return nil, err
		}
	}

	return args, nil
}

func (parser *Parser) parseNext(args []string) ([]string, error) {
	head, tail := args[0], args[1:]

	if strings.HasPrefix(head, "--") {
		name := strings.TrimPrefix(head, "--")
		return parser.handleLong(name, tail)
	}

	if strings.HasPrefix(head, "-") {
		bytes := []byte(strings.TrimPrefix(head, "-"))
		return parser.handleShort(bytes, tail)
	}

	if cmd, ok := parser.commands[head]; ok {
		program := fmt.Sprintf("%s %s", parser.program, head)
		return nil, cmd(parser.values, NewParser(program, parser.version), tail)
	}

	parser.extras = append(parser.extras, head)
	return tail, nil
}

func (parser *Parser) Parse(args []string) ([]string, error) {
	for len(args) > 0 {
		tail, err := parser.parseNext(args)
		if err != nil {
			return nil, err
		}
		args = tail
	}

	for _, p := range parser.mandatory {
		if len(parser.extras) == 0 {
			return nil, fmt.Errorf("missing mandatory argument `%s`\n%s", p.Name, parser.Usage())
		}
		p.Value.Set(parser.extras[0])
		parser.extras = parser.extras[1:]
	}

	for _, p := range parser.optional {
		if len(parser.extras) == 0 {
			return nil, nil
		}
		p.Value.Set(parser.extras[0])
		parser.extras = parser.extras[1:]
	}

	if len(parser.extras) == 0 {
		return nil, nil
	}

	return nil, fmt.Errorf("too many arguments: %s\n%s", strings.Join(parser.extras, " "), parser.Usage())
}

func wrapSpace(s string, indent int) string {
	max := 80

	if len(s) < max {
		return s
	}

	i := max - 1
	for i >= 0 && s[i] != ' ' {
		i--
	}

	if i == 0 {
		i = max - 1
		for i < len(s) && s[i] != ' ' {
			i++
		}
	}

	if i == len(s) {
		return s
	}

	t := s[:i]
	r := strings.Repeat(" ", indent-1) + s[i:]

	return t + "\n" + wrapSpace(r, indent)
}

func formatSingleArgUsage(short byte, long string, typename string) string {
	if short == 0 {
		return fmt.Sprintf("[--%s[=<%s>]]", long, typename)
	}
	return fmt.Sprintf("[-%c [<%s>]] | --%s[=<%s>]]", short, typename, long, typename)
}

func formatMultiArgUsage(short byte, long string, typename string) string {
	if short == 0 {
		return fmt.Sprintf("[--%s[=<%s>] [--%s[=<%s>] ...]]", long, typename, long, typename)
	}
	return fmt.Sprintf("[-%c [<%s> [-%c [<%s> ...]]] | --%s[=<%s>] [--%s[=<%s>] ...]]", short, typename, short, typename, long, typename, long, typename)
}

func typeUsage(short byte, long string, value Value) string {
	switch value.(type) {
	case *BoolValue:
		return formatSingleArgUsage(short, long, "bool")
	case *IntValue:
		return formatSingleArgUsage(short, long, "int")
	case *StringValue:
		return formatSingleArgUsage(short, long, "str")
	case *StringsValue:
		return formatMultiArgUsage(short, long, "str")
	default:
		panic("unknwon value type")
	}
}

func (parser Parser) argUsage(short byte, long string) string {
	if v, ok := parser.values[long]; ok {
		return typeUsage(short, long, v)
	}
	if _, ok := parser.switches[long]; ok {
		if short == 0 {
			return fmt.Sprintf("[--%s]", long)
		}
		return fmt.Sprintf("[-%c | --%s]", short, long)
	}
	panic(fmt.Errorf("value for name `%s` does not exist", long))
}

func (parser Parser) findAlias(long string) byte {
	for k, v := range parser.aliases {
		if v == long {
			return k
		}
	}
	return 0
}

func (parser Parser) names() []string {
	names := make([]string, 0)
	for k := range parser.values {
		if parser.findAlias(k) == 0 {
			names = append(names, k)
		}
	}
	for k := range parser.switches {
		if parser.findAlias(k) == 0 {
			names = append(names, k)
		}
	}
	sort.Strings(names)

	keys := []byte("0123456789AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz")

	ret := make([]string, 0)

	i := 0
	for _, short := range keys {
		if long, ok := parser.aliases[short]; ok {
			ret = append(ret, long)
		} else {
			for i < len(names) && names[i][0] == short {
				ret = append(ret, names[i])
				i++
			}
		}
	}

	return ret
}

func (parser Parser) Usage() string {
	usages := make([]string, 0)

	for _, p := range parser.mandatory {
		usages = append(usages, fmt.Sprintf("<%s>", p.Name))
	}

	for _, p := range parser.optional {
		usages = append(usages, fmt.Sprintf("[%s]", p.Name))
	}

	if len(parser.commands) > 0 {
		usages = append(usages, "<command> [args...]")
	}

	usage := strings.Join(usages, " ")
	return fmt.Sprintf("usage: %s [options...] %s", parser.program, usage)
}

func (parser Parser) isSwitch(name string) bool {
	_, ok := parser.switches[name]
	return ok
}

func (parser Parser) isSlice(name string) bool {
	if v, ok := parser.values[name]; ok {
		_, ok = v.(*StringsValue)
		return ok
	}
	return false
}

func (parser Parser) switchSyntax(short byte, long string) string {
	if short == 0 {
		return fmt.Sprintf("  --%s", long)
	}
	return fmt.Sprintf("  -%c, --%s", short, long)
}

func (parser Parser) sliceSyntax(short byte, long string) string {
	upper := strings.ToUpper(long)
	if short == 0 {
		return strings.Join([]string{
			fmt.Sprintf("  --%s <%s> [--%s <%s> ...]", long, upper, long, upper),
			fmt.Sprintf("  --%s=<%s> [--%s=<%s> ...]", long, upper, long, upper),
		}, "\n")
	}
	return strings.Join([]string{
		fmt.Sprintf("  -%c <%s> [-%c <%s> ...]", short, upper, short, upper),
		fmt.Sprintf("  --%s <%s> [--%s <%s> ...]", long, upper, long, upper),
		fmt.Sprintf("  --%s=<%s> [--%s=<%s> ...]", long, upper, long, upper),
	}, "\n")
}

func (parser Parser) argSyntax(short byte, long string) string {
	if parser.isSwitch(long) {
		return parser.switchSyntax(short, long)
	}

	if parser.isSlice(long) {
		return parser.sliceSyntax(short, long)
	}

	upper := strings.ToUpper(long)
	if short == 0 {
		return fmt.Sprintf("  --%s <%s>, --%s=<%s>", long, upper, long, upper)
	}
	return fmt.Sprintf("  -%c <%s>, --%s <%s>, --%s=<%s>", short, upper, long, upper, long, upper)
}

func (parser Parser) argDesc(long string) string {
	short := parser.findAlias(long)
	syntax := parser.argSyntax(short, long)
	usage := parser.usages[long]
	if len(syntax) > 24 {
		return fmt.Sprintf("%s\n%s%s", syntax, strings.Repeat(" ", 24), usage)
	}
	return fmt.Sprintf("%s%s%s", syntax, strings.Repeat(" ", 24-len(syntax)), usage)
}

func (parser Parser) Help() string {
	lines := []string{parser.Usage()}
	lines = append(lines, "", "optional arguments:")
	lines = append(lines, "  -h, --help            show this help message and exit")
	lines = append(lines, "  --version             show the version string and exit")
	for _, name := range parser.names() {
		lines = append(lines, parser.argDesc(name))
	}
	return strings.Join(lines, "\n")
}
