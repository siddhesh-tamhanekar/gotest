package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"golang.org/x/term"
)

var width = 80
var flag uint8
var packageCount, all, failed int
var dur float64

const (
	FAIL_ONLY = 1 << iota
	COLLAPSE
	NO_COLOR
)

// pkgInfo holds info for a package test run
type pkgInfo struct {
	tree    *TestNode
	elapsed float64
	passed  bool
	count   int
	failed  int
}

// TestNode represents a test or subtest in a tree structure
type TestNode struct {
	name     string
	children []*TestNode
	passed   bool
	elapsed  float64
	Output   string
}

// AddChild adds or returns existing child node by name
func (t *TestNode) AddChild(name string) *TestNode {
	for _, c := range t.children {
		if c.name == name {
			return c
		}
	}
	child := &TestNode{name: name}
	t.children = append(t.children, child)
	return child
}
func stripANSI(str string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(str, "")
}
func colorizeDuration(d float64) string {
	if hasFlag(NO_COLOR) {
		return fmt.Sprintf("%.2fs\x1b[0m", d)
	}
	switch {
	case d < 0.5:
		return fmt.Sprintf("\x1b[32m%.2fs\x1b[0m", d)
	case d < 2:
		return fmt.Sprintf("\x1b[33m%.2fs\x1b[0m", d)
	default:
		return fmt.Sprintf("\x1b[31m%.2fs\x1b[0m", d)
	}
}

// Print recursively prints test and subtests with indent and color
func (t *TestNode) Print(indent int) {
	var status string
	if hasFlag(FAIL_ONLY) && t.passed {
		return
	}
	if t.passed {
		status = tick()
	} else {
		status = cross()
	}
	left := fmt.Sprintf("%s%s %s ", strings.Repeat("  ", indent), status, camelCaseToSpace(t.name))
	dur := colorizeDuration(t.elapsed)
	spacing := width - len(stripANSI(left)) - len(dur)
	if spacing < 1 {
		spacing = 1
	}

	if t.name != "" {
		fmt.Printf("%s%s %s  %s\n", left, strings.Repeat(".", spacing), duration(), dur)
		if t.Output != "" {
			fmt.Printf("%s→ %s\n", strings.Repeat("  ", indent+1), t.Output)

		}
	}
	for _, c := range t.children {
		c.Print(indent + 1)
	}
}

func main() {
	// Prepare command: go test -json + user args
	forwardedArgs := parseFlags()
	args := append([]string{"test", "-json"}, forwardedArgs...)
	cmd := exec.Command("go", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(stdout)
	// Map package name -> package info (test tree, elapsed, pass/fail)
	pkgs := map[string]*pkgInfo{}
	width, _, _ = term.GetSize(int(os.Stdout.Fd()))
	if width == 0 {
		width = 80
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		var m map[string]interface{}
		if err := json.Unmarshal(line, &m); err != nil {
			continue // skip lines that aren't valid JSON
		}

		action, _ := m["Action"].(string)
		testName, _ := m["Test"].(string)
		pkgName, _ := m["Package"].(string)
		if testName == "" && pkgName == "" {
			continue
		}
		// Initialize package info if not present
		if _, ok := pkgs[pkgName]; !ok {
			pkgs[pkgName] = &pkgInfo{
				tree:   &TestNode{name: ""},
				passed: true,
			}
		}

		pkg := pkgs[pkgName]
		if action == "output" {
			opText := m["Output"].(string)
			if strings.HasPrefix(opText, "---") || strings.HasPrefix(opText, "===") {
				continue
			}
			parts := strings.Split(testName, "/")

			curr := pkg.tree
			for _, part := range parts {
				curr = curr.AddChild(part)
			}
			curr.Output = strings.Trim(opText, " \n")
		}
		if action == "pass" || action == "fail" {
			if testName != "" {
				// Handle test and subtest path, separated by /
				parts := strings.Split(testName, "/")
				pkg.count += 1
				curr := pkg.tree
				for _, part := range parts {
					curr = curr.AddChild(part)
				}
				curr.passed = (action == "pass")
				if !curr.passed {
					pkg.failed += 1
				}

				if elapsed, ok := m["Elapsed"].(float64); ok {
					curr.elapsed = elapsed
				}
				// If any test fails, mark package as failed
				if action == "fail" {
					pkg.passed = false
				}
			} else {
				// Package summary line
				if elapsed, ok := m["Elapsed"].(float64); ok {
					pkg.elapsed = elapsed
				}
				// When package completes, print all tests
				var status string
				if action == "pass" {
					status = tick()
				} else {
					status = cross()
				}
				fmt.Printf("%s Package: %s   "+tick()+" Passed:%d   "+cross()+" Failed: %d   "+duration()+"  Duration: %.2fs\n", status, pkgName, pkg.count-pkg.failed, pkg.failed, pkg.elapsed)
				all += pkg.count
				failed += pkg.failed
				dur += pkg.elapsed
				packageCount++
				if !hasFlag(COLLAPSE) {
					pkg.tree.Print(1)
					fmt.Println()
				}

				// Delete package info to free memory and avoid reprinting
				delete(pkgs, pkgName)
			}
		}
	}
	status := tick()
	if failed > 0 {
		status = cross()
	}
	fmt.Println(strings.Repeat("=", width))
	fmt.Printf("%s  Total Packages: %d   "+tick()+" Passed:%d   "+cross()+" Failed: %d   "+duration()+"  Duration: %.2fs\n", status, packageCount, all-failed, failed, dur)
	fmt.Println(strings.Repeat("=", width))

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	if err := cmd.Wait(); err != nil {
		// ignore: go test -json returns non-zero on test failure
	}
}

func parseFlags() []string {
	forwardedArgs := []string{}
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--fail-only":
			flag = flag | FAIL_ONLY
		case "--collapse":
			flag = flag | COLLAPSE
		case "--no-color":
			flag = flag | NO_COLOR

		default:
			forwardedArgs = append(forwardedArgs, arg)
		}
	}
	if hasFlag(FAIL_ONLY) && hasFlag(COLLAPSE) {
		fmt.Println("NOTE: fail-only flag has no use when passed with collapsed")
	}
	return forwardedArgs
}

// camelCaseToSpace inserts spaces before uppercase letters, trims "Test " prefix
func camelCaseToSpace(s string) string {
	re := regexp.MustCompile(`([a-z])([A-Z])(\_)`)
	s1 := re.ReplaceAllString(s, `$1 $2`)
	s1 = strings.ReplaceAll(s1, "_", " ")
	s1 = strings.ReplaceAll(s1, "-", " ")

	return strings.TrimPrefix(s1, "Test ")
}

func tick() string {
	if hasFlag(NO_COLOR) {
		return "✓"
	}
	return "\x1b[32m✓\x1b[0m"
}
func cross() string {
	if hasFlag(NO_COLOR) {
		return "✘"
	}
	return "\x1b[31m✘\x1b[0m"
}
func duration() string {
	return "⏱"
}

func hasFlag(f byte) bool {
	return flag&f != 0
}
