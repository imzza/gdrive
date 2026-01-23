package handlers

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/imzza/gdrive/internal/cli"
	"github.com/imzza/gdrive/internal/utils"
)

var AppName string
var AppVersion string

func PrintVersion(ctx cli.Context) {
	fmt.Printf("%s: %s\n", AppName, AppVersion)
	fmt.Printf("Golang: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

func PrintHelp(ctx cli.Context) {
	root := buildCommandTree(ctx.Handlers())
	printCommandList([]string{}, root, true)
}

func PrintCommandHelp(ctx cli.Context) {
	args := ctx.Args()
	printScopedHelp(ctx, []string{args.String("command")})
}

func PrintSubCommandHelp(ctx cli.Context) {
	args := ctx.Args()
	printScopedHelp(ctx, []string{args.String("command"), args.String("subcommand")})
}

func PrintSubSubCommandHelp(ctx cli.Context) {
	args := ctx.Args()
	printScopedHelp(ctx, []string{args.String("command"), args.String("subcommand"), args.String("subsubcommand")})
}

func DrivesHelpHandler(ctx cli.Context) {
	printScopedHelp(ctx, []string{"drives"})
}

func AccountHelpHandler(ctx cli.Context) {
	printScopedHelp(ctx, []string{"account"})
}

func FilesHelpHandler(ctx cli.Context) {
	printScopedHelp(ctx, []string{"files"})
}

func PermissionsHelpHandler(ctx cli.Context) {
	printScopedHelp(ctx, []string{"permissions"})
}

func FilesSubcommandHelpHandler(ctx cli.Context) {
	args := ctx.Args()
	printCommandPrefixHelp(ctx, "files", args.String("subcommand"))
}

func FilesSyncHelpHandler(ctx cli.Context) {
	printScopedHelp(ctx, []string{"files", "sync"})
}

func printCommandPrefixHelp(ctx cli.Context, prefix ...string) {
	handler := getHandler(ctx.Handlers(), prefix)

	if handler == nil {
		utils.ExitF("Command not found")
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 3, ' ', 0)

	fmt.Fprintf(w, "%s\n", handler.Description)
	fmt.Fprintf(w, "%s %s\n", AppName, handler.Pattern)
	for _, group := range handler.FlagGroups {
		fmt.Fprintf(w, "\n%s:\n", group.Name)
		for _, flag := range group.Flags {
			boolFlag, isBool := flag.(cli.BoolFlag)
			if isBool && boolFlag.OmitValue {
				fmt.Fprintf(w, "  %s\t%s\n", strings.Join(flag.GetPatterns(), ", "), flag.GetDescription())
			} else {
				fmt.Fprintf(w, "  %s <%s>\t%s\n", strings.Join(flag.GetPatterns(), ", "), flag.GetName(), flag.GetDescription())
			}
		}
	}

	w.Flush()
}

func printScopedHelp(ctx cli.Context, prefix []string) {
	root := buildCommandTree(ctx.Handlers())
	node := findCommandNode(root, prefix)
	if node == nil {
		utils.ExitF("Command not found")
	}

	if len(node.children) == 0 {
		printCommandPrefixHelp(ctx, prefix...)
		return
	}

	printCommandList(prefix, node, false)
}

func printCommandList(prefix []string, node *commandNode, includeOptions bool) {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 3, ' ', 0)

	if len(prefix) == 0 {
		fmt.Fprintf(w, "Usage: %s <COMMAND>\n\n", AppName)
	} else {
		fmt.Fprintf(w, "Usage: %s %s <COMMAND>\n\n", AppName, strings.Join(prefix, " "))
	}

	fmt.Fprintln(w, "Commands:")

	order := commandOrder(prefix)
	keys := orderedKeys(node.children, order)
	if len(prefix) > 0 && !containsKey(node.children, "help") {
		keys = append(keys, "help")
	}
	for _, name := range keys {
		if strings.HasPrefix(name, "-") {
			continue
		}
		desc := ""
		if child, ok := node.children[name]; ok && child != nil {
			desc = child.desc
		}
		if len(prefix) == 0 {
			if topDesc := topLevelDescription(name); topDesc != "" {
				desc = topDesc
			}
		}
		if desc == "" {
			desc = groupDescription(name)
		}
		if desc == "" && name == "help" {
			desc = topLevelDescription("help")
		}
		if desc == "" {
			fmt.Fprintf(w, "  %s\n", name)
		} else {
			fmt.Fprintf(w, "  %s\t%s\n", name, desc)
		}
	}

	if includeOptions {
		fmt.Fprintln(w, "\nOptions:")
		fmt.Fprintln(w, "  -h, --help\tPrint help information")
	}

	w.Flush()
}

type commandNode struct {
	desc     string
	children map[string]*commandNode
}

func buildCommandTree(handlers []*cli.Handler) *commandNode {
	root := &commandNode{children: map[string]*commandNode{}}
	for _, h := range handlers {
		tokens := literalTokens(h)
		if len(tokens) == 0 {
			continue
		}
		addCommandPath(root, tokens, h.Description)
	}
	return root
}

func addCommandPath(root *commandNode, tokens []string, desc string) {
	node := root
	for i, token := range tokens {
		if node.children == nil {
			node.children = map[string]*commandNode{}
		}
		child := node.children[token]
		if child == nil {
			child = &commandNode{children: map[string]*commandNode{}}
			node.children[token] = child
		}
		node = child
		if i == len(tokens)-1 && node.desc == "" {
			node.desc = desc
		}
	}
}

func literalTokens(h *cli.Handler) []string {
	var tokens []string
	for _, token := range h.SplitPattern() {
		if isFlagGroupToken(token) || isCaptureGroupToken(token) {
			continue
		}
		if token == "-" {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens
}

func findCommandNode(root *commandNode, prefix []string) *commandNode {
	node := root
	for _, token := range prefix {
		if node == nil {
			return nil
		}
		node = node.children[token]
	}
	return node
}

func isCaptureGroupToken(arg string) bool {
	return strings.HasPrefix(arg, "<") && strings.HasSuffix(arg, ">")
}

func isFlagGroupToken(arg string) bool {
	return strings.HasPrefix(arg, "[") && strings.HasSuffix(arg, "]")
}

func orderedKeys(children map[string]*commandNode, order []string) []string {
	seen := map[string]bool{}
	var keys []string
	for _, name := range order {
		if _, ok := children[name]; ok {
			keys = append(keys, name)
			seen[name] = true
		}
	}
	extraStart := len(keys)
	for name := range children {
		if !seen[name] {
			keys = append(keys, name)
		}
	}
	if len(order) == 0 {
		sort.Strings(keys)
	} else {
		sort.Strings(keys[extraStart:])
	}
	return keys
}

func containsKey(children map[string]*commandNode, name string) bool {
	_, ok := children[name]
	return ok
}

func commandOrder(prefix []string) []string {
	if len(prefix) == 0 {
		return []string{"about", "account", "drives", "files", "permissions", "version", "help"}
	}

	switch prefix[len(prefix)-1] {
	case "account":
		return []string{"add", "list", "current", "switch", "remove", "export", "import"}
	case "files":
		return []string{"list", "download", "upload", "update", "info", "mkdir", "rename", "move", "copy", "delete", "import", "export", "changes", "sync", "revision"}
	case "permissions":
		return []string{"share", "list", "revoke"}
	case "drives":
		return []string{"list"}
	case "sync":
		return []string{"list", "content", "download", "upload"}
	case "revision":
		return []string{"list", "download", "delete"}
	default:
		return nil
	}
}

func groupDescription(name string) string {
	switch name {
	case "sync":
		return "Commands for syncing files"
	case "revision":
		return "Commands for managing file revisions"
	default:
		return ""
	}
}

func topLevelDescription(name string) string {
	switch name {
	case "about":
		return "Print information about gdrive"
	case "account":
		return "Commands for managing accounts"
	case "drives":
		return "Commands for managing drives"
	case "files":
		return "Commands for managing files"
	case "permissions":
		return "Commands for managing file permissions"
	case "version":
		return "Print version information"
	case "help":
		return "Print this message or the help of the given subcommand(s)"
	default:
		return ""
	}
}

func getHandler(handlers []*cli.Handler, prefix []string) *cli.Handler {
	for _, h := range handlers {
		pattern := stripOptionals(h.SplitPattern())

		if len(prefix) > len(pattern) {
			continue
		}

		if utils.Equal(prefix, pattern[:len(prefix)]) {
			return h
		}
	}

	return nil
}

// Strip optional groups (<...>) from pattern
func stripOptionals(pattern []string) []string {
	newArgs := []string{}

	for _, arg := range pattern {
		if strings.HasPrefix(arg, "[") && strings.HasSuffix(arg, "]") {
			continue
		}
		newArgs = append(newArgs, arg)
	}
	return newArgs
}
