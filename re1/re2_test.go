package re1

// These run tests from the RE2 test suite,
// filtering out syntax re1 doesn't support.
// We ignore the match info in the RE2 test suite,
// because it's for single-line mode and first match.
// re1 is always multi-line mode and longest match.
// Instead, we test against the Go regexp package with
// both multi-line mode and longest matching enabled.
// We ignore substring matches;
// regexp and RE2 compute different submatches than re1.

import (
	"bufio"
	"compress/bzip2"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/eaburns/T/rope"
)

func TestRE2Search(t *testing.T) {
	runRE2Tests(t, "testdata/re2-search.txt.bz2")
}

func TestRE2Exhaustive(t *testing.T) {
	runRE2Tests(t, "testdata/re2-exhaustive.txt.bz2")
}

func runRE2Tests(t *testing.T, path string) {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer f.Close()

	var strs []string
	var inStrings bool
	scanner := bufio.NewScanner(bzip2.NewReader(f))
	for scanner.Scan() {
		switch line := strings.TrimSpace(scanner.Text()); {
		case line == "strings":
			strs = strs[:0]
			inStrings = true
		case line == "regexps":
			inStrings = false
		case inStrings:
			strs = append(strs, mustUnquote(line))
		default:
			reStr, err := strconv.Unquote(line)
			if err != nil {
				// Ignore re2 match info; it's neither multi-line more nor longest.
				// Instead we compare to the go regexp directly.
				continue
			}
			if unsupported(reStr) {
				continue
			}
			runRE2TestCase(t, reStr, strs)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf(err.Error())
	}
}

func runRE2TestCase(t *testing.T, reStr string, strs []string) {
	reStr = strings.Replace(reStr, "(?:", "(", -1)
	goRegexp := regexp.MustCompile("(?m:" + reStr + ")")
	goRegexp.Longest()
	r1Regexp, residue, err := New(reStr, Opts{})
	if residue != "" || err != nil {
		t.Errorf("New(%q, '/')=_,%q,%v", reStr, residue, err)
		return
	}
	for _, str := range strs {
		want := match64(goRegexp, str)
		got := r1Regexp.Find(strings.NewReader(str))
		// We only consider the full match,
		// because we disagree with regexp and re2
		// on what the submatches are.
		if got != nil {
			got = got[:2]
		}
		if !reflect.DeepEqual(want, got) {
			t.Errorf("%q: Find(%q)=%v, want %v", reStr, str, got, want)
		}

		ro := rope.New(str)
		got = r1Regexp.FindInRope(ro, 0, ro.Len())
		if got != nil {
			got = got[:2]
		}
		if !reflect.DeepEqual(want, got) {
			t.Errorf("%q: FindInRope(%q)=%v, want %v", reStr, str, got, want)
		}
	}

}

func match64(re *regexp.Regexp, str string) []int64 {
	var ms []int64
	for _, m := range re.FindStringIndex(str) {
		ms = append(ms, int64(m))
	}
	return ms
}

var (
	exclude = []string{
		// We don't support flags.
		`(?i`,
		`(?m`,
		`(?s`,
		`(?u`,

		// We don't support non-greedy repetition.
		`*?`,
		`+?`,
		`??`,

		// We don't support [[:space:]] and friends.
		`[[`,

		// We don't support these Perl character classes.
		`\A`,
		`\B`,
		`\C`,
		`\D`,
		`\P`,
		`\S`,
		`\W`,
		`\a`,
		`\b`,
		`\d`,
		`\f`,
		`\p`,
		`\r`,
		`\s`,
		`\S`,
		`\v`,
		`\w`,
		`\x`,
		`\z`,

		// We have a less-permissive charclass grammar.
		`[]a]`,
		`[-a]`,
		`[a-]`,
		`[^-a]`,
		`[a-b-c]`,

		// We don't support octal.
		`\608`,
		`\01`,
		`\018`,
	}

	octal = regexp.MustCompile(`[\][0-7][0-7][0-7]`)
	repN  = regexp.MustCompile(`\{[0-9,]+\}`)
)

func unsupported(re string) bool {
	for _, x := range exclude {
		if strings.Contains(re, x) {
			return true
		}
	}
	return octal.MatchString(re) || repN.MatchString(re)
}

func parseMatches(line string) [][]int64 {
	var matches [][]int64
	for _, match := range strings.Split(line, ";") {
		var ms []int64
		if match == "-" {
			matches = append(matches, ms)
			continue
		}
		for _, subMatch := range strings.Split(match, " ") {
			if subMatch == "-" {
				ms = append(ms, -1, -1)
				continue
			}
			fs := strings.Split(subMatch, "-")
			if len(fs) != 2 {
				panic("bad submatch range")
			}
			ms = append(ms, atoi(fs[0]), atoi(fs[1]))
		}
		matches = append(matches, ms)
	}
	return matches
}

func atoi(str string) int64 {
	i, err := strconv.Atoi(str)
	if err != nil {
		panic("bad int " + err.Error())
	}
	return int64(i)
}

func mustUnquote(line string) string {
	line, err := strconv.Unquote(line)
	if err != nil {
		panic(err.Error())
	}
	return line
}
