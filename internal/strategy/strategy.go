package strategy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	ErrWinWSCommandNotFound = errors.New("winws.exe command not found")
	ErrArgsQuoteMismatch    = errors.New("mismatched quotes in strategy args")
	ErrNotFound             = errors.New("no strategies found")
)

type Strategy struct {
	Name   string
	Path   string
	Custom bool
}

type ParsedStrategy struct {
	Strategy Strategy
	Args     []string
}

type GameFilterPorts struct {
	All string
	TCP string
	UDP string
}

func List(root string, customNames []string, extraDir string) ([]Strategy, error) {
	customSet := buildCustomSet(customNames)

	regular, custom, err := collectFromDir(root, customSet, false)
	if err != nil {
		return nil, fmt.Errorf("failed to read dir: %w", err)
	}

	_, extraCustom, err := collectFromDir(extraDir, nil, true)
	if err != nil {
		return nil, fmt.Errorf("failed to read custom strategies dir: %w", err)
	}
	custom = append(custom, extraCustom...)

	sortByName(regular)
	sortByName(custom)

	return append(regular, custom...), nil
}

func buildCustomSet(names []string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[strings.ToLower(n)] = struct{}{}
	}
	return set
}

func collectFromDir(dir string, customSet map[string]struct{}, forceCustom bool) ([]Strategy, []Strategy, error) {
	if dir == "" {
		return nil, nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to read dir %s: %w", dir, err)
	}
	var regular, custom []Strategy
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".bat") {
			continue
		}
		name := entry.Name()
		if !forceCustom && strings.HasPrefix(strings.ToLower(name), "service") {
			continue
		}
		s := Strategy{
			Name:   name,
			Path:   filepath.Join(dir, name),
			Custom: forceCustom || isCustom(name, customSet),
		}
		if s.Custom {
			custom = append(custom, s)
		} else {
			regular = append(regular, s)
		}
	}
	return regular, custom, nil
}

func sortByName(ss []Strategy) {
	sort.Slice(ss, func(i, j int) bool {
		return strings.ToLower(ss[i].Name) < strings.ToLower(ss[j].Name)
	})
}

func isCustom(name string, customSet map[string]struct{}) bool {
	_, ok := customSet[strings.ToLower(name)]
	return ok
}

func Parse(s Strategy, root string, ports GameFilterPorts) (*ParsedStrategy, error) {
	content, err := os.ReadFile(s.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read strategy file: %w", err)
	}

	command, ok := findWinWSCommand(string(content))
	if !ok {
		return nil, fmt.Errorf("failed to find winws.exe command in strategy file: %w", ErrWinWSCommandNotFound)
	}

	args, err := parseWinWSArgs(command, root, ports)
	if err != nil {
		return nil, fmt.Errorf("failed to parse winws.exe args: %w", err)
	}

	return &ParsedStrategy{
		Strategy: s,
		Args:     args,
	}, nil
}

func findWinWSCommand(content string) (string, bool) {
	var commands []string
	var current strings.Builder

	for _, rawLine := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "::") || strings.EqualFold(line, "rem") || strings.HasPrefix(strings.ToLower(line), "rem ") {
			continue
		}

		continued := strings.HasSuffix(line, "^")
		if continued {
			line = strings.TrimSpace(strings.TrimSuffix(line, "^"))
		}

		if current.Len() > 0 {
			current.WriteByte(' ')
		}
		current.WriteString(line)

		if continued {
			continue
		}

		commands = append(commands, current.String())
		current.Reset()
	}

	if current.Len() > 0 {
		commands = append(commands, current.String())
	}

	for _, command := range commands {
		if strings.Contains(strings.ToLower(command), "winws.exe") {
			return strings.ReplaceAll(command, "^!", "!"), true
		}
	}

	return "", false
}

func parseWinWSArgs(command, root string, ports GameFilterPorts) ([]string, error) {
	const executable = "winws.exe"

	index := strings.Index(strings.ToLower(command), executable)
	if index < 0 {
		return nil, ErrWinWSCommandNotFound
	}

	rawArgs := strings.TrimSpace(command[index+len(executable):])
	rawArgs = strings.TrimPrefix(rawArgs, `"`)

	binPath := filepath.Join(root, "bin") + string(os.PathSeparator)
	listsPath := filepath.Join(root, "lists") + string(os.PathSeparator)

	rawArgs = strings.NewReplacer(
		"%BIN%", binPath,
		"%LISTS%", listsPath,
		"%GameFilter%", ports.All,
		"%GameFilterTCP%", ports.TCP,
		"%GameFilterUDP%", ports.UDP,
	).Replace(rawArgs)

	parsedArgs := make([]string, 0, strings.Count(rawArgs, " ")+1)
	var current strings.Builder
	inQuotes := false

	for _, r := range rawArgs {
		switch r {
		case '"':
			inQuotes = !inQuotes
		case ' ', '\t':
			if inQuotes {
				current.WriteRune(r)
				continue
			}
			if current.Len() > 0 {
				parsedArgs = append(parsedArgs, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if inQuotes {
		return nil, ErrArgsQuoteMismatch
	}
	if current.Len() > 0 {
		parsedArgs = append(parsedArgs, current.String())
	}

	return parsedArgs, nil
}
