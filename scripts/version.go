// run as 'go run ./scripts/version.go'

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"text/template"

	"github.com/Masterminds/semver"
)

func logAndExit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Outputs the current version number for this repo\nUsage:\n")
		flag.PrintDefaults()
	}
	goflagsVar := flag.Bool("g", false, "generate Go flags instead of just version")
	next := flag.Bool("next", false, "list the next version for tagging")
	releaseTag := flag.Bool("release", false, "tag a new release, and push it")
	nosign := flag.Bool("nosign", false, "don't sign release tags (use with -release)")
	flag.Parse()

	ver, err := version()
	logAndExit(err)

	switch {
	case *next:
		nextVer := ver.IncPatch()
		fmt.Println((&nextVer).String())
	case *releaseTag:
		err := release(ver, *nosign)
		logAndExit(err)
	case !*goflagsVar:
		fmt.Printf("%s\n", ver)
	default:
		// we use the prometheus/common/version package to store the version
		// for simplicity when exposing the build_info metric
		prefix := "github.com/prometheus/common/version"
		branch, err := runCmd("git rev-parse --abbrev-ref HEAD")
		logAndExit(err)
		revision, err := runCmd("git rev-parse --short HEAD")
		logAndExit(err)
		user, err := user.Current()
		logAndExit(err)
		hostname, err := os.Hostname()
		logAndExit(err)

		tmpl, err := template.New("t").Parse(
			"-X {{.prefix}}.Branch={{.branch}} " +
				"-X {{.prefix}}.Version={{.version}} " +
				"-X {{.prefix}}.Revision={{.revision}} " +
				"-X {{.prefix}}.BuildUser={{.user}}@{{.hostname}} ",
		)
		logAndExit(err)

		vals := map[string]string{
			"prefix":   prefix,
			"branch":   branch,
			"version":  ver.String(),
			"revision": revision,
			"user":     user.Username,
			"hostname": hostname,
		}

		out := &bytes.Buffer{}
		err = tmpl.Execute(out, vals)
		logAndExit(err)

		fmt.Println(out.String())
	}
}

// runCmd runs the given command and returns the combined output and/or an error
func runCmd(c string) (string, error) {
	parts := strings.Split(c, " ")
	//nolint:gosec
	cmd := exec.Command(parts[0], parts[1:]...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// version returns the correct version string to use given the current state of
// this repo
func version() (*semver.Version, error) {
	desc, err := runCmd("git describe --always --tags --dirty")
	if err != nil {
		return nil, fmt.Errorf("git describe failed: %w", err)
	}

	return parseVersion(desc)
}

// parseVersion parses a version string as returned by 'git describe --always',
// and if it's not on a tag, increments the patch so it's a valid prerelease
func parseVersion(in string) (*semver.Version, error) {
	ver, err := semver.NewVersion(in)
	if err != nil {
		return nil, err
	}

	// if we're not on a tag, use the bits that git appended to the tag as the
	// prerelease, but first increment the patch
	// no need to check error since we already know this is an integer
	v2 := *ver
	if v2.Prerelease() != "" {
		// one call just removes the prerelease part, so we need a second call
		v2 = v2.IncPatch().IncPatch()
	} else if v2.Metadata() != "" {
		// if only metadata is provided, IncPatch() increments the patch and
		// removes the metadata part
		v2 = v2.IncPatch()
	}

	// no-op on releases
	v2, _ = v2.SetPrerelease(ver.Prerelease())
	v2, _ = v2.SetMetadata(ver.Metadata())

	return &v2, nil
}

func release(ver *semver.Version, nosign bool) error {
	// make sure this is being run from main
	out, err := runCmd("git status -sb -u no")
	if err != nil {
		return err
	}

	if strings.TrimSpace(out) != "## main...origin/main" {
		return fmt.Errorf("must release from the main branch - check it out and pull first")
	}

	// make sure we're up to date
	out, err = runCmd("git pull")
	if err != nil {
		return err
	}

	if strings.TrimSpace(out) != "Already up to date." {
		fmt.Printf("git pull:\n%s\n", out)
	}

	nextVer := ver
	*nextVer = ver.IncPatch()

	// default to GPG-signed tags, unless -nosign was set
	tagArgs := "-sm"
	if nosign {
		tagArgs = "-am"
	}

	cmd := fmt.Sprintf(`git tag %s "v%s" v%s`, tagArgs, nextVer, nextVer)
	out, err = runCmd(cmd)
	fmt.Println(out)
	if err != nil {
		return fmt.Errorf("failed to run cmd %s: %w", cmd, err)
	}

	cmd = fmt.Sprintf(`git push origin v%s`, nextVer)
	out, err = runCmd(cmd)
	fmt.Println(out)
	if err != nil {
		return fmt.Errorf("failed to run cmd %s: %w", cmd, err)
	}

	fmt.Printf("v%s is tagged and pushed!\n", nextVer)

	ghReleaseCmd := fmt.Sprintf("gh release create v%s --generate-notes", nextVer)
	out, err = runCmd(ghReleaseCmd)
	fmt.Println(out)
	if err != nil {
		if strings.Contains(err.Error(), `"gh": executable file not found in $PATH`) {
			return fmt.Errorf("gh is not installed, please install via https://cli.github.com/")
		}
		return fmt.Errorf("failed to run cmd %s: %w", ghReleaseCmd, err)
	}

	fmt.Printf("v%s release is created on GitHub\n", nextVer)

	return nil
}
