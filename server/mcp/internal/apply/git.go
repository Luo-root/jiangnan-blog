package apply

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type LogEntry struct {
	SHA     string `json:"sha"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Subject string `json:"subject"`
}

func Head(gitDir string) string {
	if gitDir == "" {
		return ""
	}
	out, err := runGit(gitDir, "", "rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	return out
}

func Show(gitDir, commit, rel string) (string, error) {
	spec := commit + ":" + filepath.ToSlash(rel)
	cmd := exec.Command("git", "--git-dir="+gitDir, "show", spec)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func Commit(gitDir, workTree, rel, msg string) (string, error) {
	if gitDir == "" {
		return "", nil
	}
	if _, err := runGit(gitDir, workTree, "add", "--", rel); err != nil {
		return "", err
	}
	out, err := runGit(gitDir, workTree,
		"-c", "user.name=workbase",
		"-c", "user.email=workbase@local",
		"commit", "-m", msg)
	if err != nil {
		if strings.Contains(out, "nothing to commit") {
			return Head(gitDir), nil
		}
		return "", err
	}
	return Head(gitDir), nil
}

func Restore(gitDir, workTree, rel string) {
	if gitDir == "" {
		return
	}
	_, _ = runGit(gitDir, workTree, "checkout", "HEAD", "--", rel)
}

func MergeFile(ours, base, other string) (merged string, conflict bool, err error) {
	dir, err := os.MkdirTemp("", "workbase-merge-")
	if err != nil {
		return "", false, err
	}
	defer os.RemoveAll(dir)

	oursPath := filepath.Join(dir, "ours")
	basePath := filepath.Join(dir, "base")
	otherPath := filepath.Join(dir, "other")
	if err := os.WriteFile(oursPath, []byte(ours), 0644); err != nil {
		return "", false, err
	}
	if err := os.WriteFile(basePath, []byte(base), 0644); err != nil {
		return "", false, err
	}
	if err := os.WriteFile(otherPath, []byte(other), 0644); err != nil {
		return "", false, err
	}

	cmd := exec.Command("git", "merge-file", "-p", oursPath, basePath, otherPath)
	out, err := cmd.Output()
	text := string(out)
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return text, true, nil
		}
		return "", false, err
	}
	if strings.Contains(text, "<<<<<<<") {
		return text, true, nil
	}
	return text, false, nil
}

func Log(gitDir string, limit int) ([]LogEntry, error) {
	if gitDir == "" {
		return []LogEntry{}, nil
	}
	if limit <= 0 {
		limit = 30
	}
	if limit > 200 {
		limit = 200
	}
	out, err := runGit(gitDir, "", "-c", "core.quotepath=false", "log", "-n", strconv.Itoa(limit), "--pretty=format:%H%x09%an%x09%aI%x09%s")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return []LogEntry{}, nil
	}
	lines := strings.Split(out, "\n")
	commits := make([]LogEntry, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		commits = append(commits, LogEntry{SHA: parts[0], Author: parts[1], Date: parts[2], Subject: parts[3]})
	}
	return commits, nil
}

func Patch(gitDir, commit string) (string, error) {
	if gitDir == "" || commit == "" {
		return "", os.ErrInvalid
	}
	return runGit(gitDir, "", "-c", "core.quotepath=false", "show", "--stat", "--patch", "--format=fuller", "--no-color", commit)
}

func runGit(gitDir, workTree string, args ...string) (string, error) {
	all := make([]string, 0, len(args)+4)
	if gitDir != "" {
		all = append(all, "--git-dir="+gitDir)
	}
	if workTree != "" {
		all = append(all, "--work-tree="+workTree)
	}
	all = append(all, args...)
	cmd := exec.Command("git", all...)
	if workTree != "" {
		cmd.Dir = workTree
	}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
