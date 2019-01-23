package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	cosPkgPath = "github.com/coschain/contentos-go"
)

type statements struct {
	BeginLine, BeginCol, EndLine, EndCol int
	Count int
}

type coverage struct {
	statements
	Cover int
}

type coverageResult struct {
	mode string
	cov map[string][]coverage
}

func srcFiles(root string, excludes []string, includes []string) (files []string) {
	root, _ = filepath.Abs(root)
	var exc, inc []string
	pathSep := string(filepath.Separator)
	for _, s := range excludes {
		ss := filepath.Join(root, s)
		if strings.HasSuffix(s, pathSep) && !strings.HasSuffix(ss, pathSep) {
			ss = ss + pathSep
		}
		exc = append(exc, ss)
	}
	for _, s := range includes {
		ss := filepath.Join(root, s)
		if strings.HasSuffix(s, pathSep) && !strings.HasSuffix(ss, pathSep) {
			ss = ss + pathSep
		}
		inc = append(inc, ss)
	}
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			name := info.Name()
			if info.IsDir() {
				if strings.HasPrefix(name, ".") || name == "testdata" {
					return filepath.SkipDir
				}
			} else if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") && !strings.HasSuffix(name, ".pb.go") {
				ignore := false
				for _, s := range exc {
					if strings.HasPrefix(path, s) {
						ignore = true
						break
					}
				}
				if ignore {
					for _, s := range inc {
						if strings.HasPrefix(path, s) {
							ignore = false
							break
						}
					}
				}
				if !ignore {
					files = append(files, path)
				}
			}
		}
		return nil
	})
	return files
}

func countStatements(fileSet *token.FileSet, file *ast.File) (s []statements) {
	for _, d := range file.Decls {
		if f, isFuncDecl := d.(*ast.FuncDecl); isFuncDecl {
			if f.Body != nil {
				if sc := len(f.Body.List); sc > 0 {
					s0, s1 := f.Body.List[0], f.Body.List[sc - 1]
					p0, p1 := fileSet.Position(s0.Pos()), fileSet.Position(s1.End())
					s = append(s, statements{
						BeginLine: p0.Line,
						BeginCol: p0.Column,
						EndLine: p1.Line,
						EndCol: p1.Column,
						Count: sc,
					})
				}
			}
		}
	}
	return s
}

func parseFiles(files []string, root string) *coverageResult {
	cov := make(map[string][]coverage)
	fs := token.NewFileSet()
	for _, file := range files {
		f, err := parser.ParseFile(fs, file, nil, 0)
		if err == nil && f != nil {
			base, _ := filepath.Rel(root, file)
			fn := filepath.Join(cosPkgPath, base)
			s := countStatements(fs, f)
			cov[fn] = make([]coverage, len(s))
			for i := range s {
				cov[fn][i] = coverage{s[i], 0}
			}
		}
	}
	return &coverageResult{ "set", cov }
}

func parseCoverage(file string) *coverageResult {
	cov := &coverageResult{ "set", make(map[string][]coverage) }
	if f, err := os.Open(file); err == nil {
		defer f.Close()
		reader := bufio.NewReader(f)
		count := 0
		delim := regexp.MustCompile(`[.,\s]+`)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			line = strings.Trim(line, " \t\r\n")
			if len(line) == 0 {
				continue
			}
			count++
			if count == 1 {
				if strings.HasPrefix(line, "mode:") {
					cov.mode = strings.Trim(line[5:], " \t\r\n")
				} else {
					break
				}
			} else {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					fn := parts[0]
					args := delim.Split(parts[1], -1)
					if len(args) == 6 {
						l0, _ := strconv.Atoi(args[0])
						c0, _ := strconv.Atoi(args[1])
						l1, _ := strconv.Atoi(args[2])
						c1, _ := strconv.Atoi(args[3])
						sc, _ := strconv.Atoi(args[4])
						cc, _ := strconv.Atoi(args[5])
						cov.cov[fn] = append(cov.cov[fn], coverage{
							statements: statements{
								BeginLine: l0,
								BeginCol: c0,
								EndLine: l1,
								EndCol: c1,
								Count: sc,
							},
							Cover: cc,
						})
					}
				}
			}
		}
	}
	return cov
}

func dumpCoverage(cov *coverageResult) {
	fmt.Printf("mode: %s\n", cov.mode)
	for file, covs := range cov.cov {
		for _, c := range covs {
			fmt.Printf("%s:%d.%d,%d.%d %d %d\n",
				file,
				c.BeginLine, c.BeginCol,
				c.EndLine, c.EndCol,
				c.Count, c.Cover)
		}
	}
}

func totalCoverage(empty, tested *coverageResult) *coverageResult {
	merged := &coverageResult{ mode: tested.mode, cov: tested.cov }
	for file := range empty.cov {
		if _, has := merged.cov[file]; has {
			continue
		}
		merged.cov[file] = empty.cov[file]
	}
	return merged
}

var (
	skips = []string {
		"utils/",
		"app/table/",
		"db/table/",
		"cmd/multinodetester/",
		"cmd/multinodetester2/",
	}
	skipExcepts = []string {}
)

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s <src_root_dir> <coverage_profile>\n", filepath.Base(os.Args[0]))
	} else {
		root, _ := filepath.Abs(os.Args[1])
		testCov, _ := filepath.Abs(os.Args[2])
		dumpCoverage(totalCoverage(parseFiles(srcFiles(root, skips, skipExcepts), root), parseCoverage(testCov)))
	}
}
