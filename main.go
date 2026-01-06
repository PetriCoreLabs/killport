package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// version is set via ldflags during build
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var port int
	var autoConfirm bool

	// Parse arguments
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "-h", "--help":
			printUsage()
			return
		case "-v", "--version":
			fmt.Printf("killport v%s\n", version)
			return
		case "-y", "--yes":
			autoConfirm = true
		default:
			// Try to parse as port number
			p, err := strconv.Atoi(arg)
			if err != nil || p < 1 || p > 65535 {
				fmt.Fprintf(os.Stderr, "Error: Invalid port number '%s'. Port must be between 1 and 65535.\n", arg)
				os.Exit(1)
			}
			port = p
		}
	}

	if port == 0 {
		fmt.Fprintf(os.Stderr, "Error: No port specified.\n")
		printUsage()
		os.Exit(1)
	}

	// First, find what's running on the port
	processInfo, pids, err := findProcessOnPort(port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(pids) == 0 {
		fmt.Printf("No process found running on port %d\n", port)
		return
	}

	// Show what we found
	fmt.Printf("Found process(es) on port %d:\n", port)
	fmt.Println(processInfo)

	// Confirm before killing (unless -y flag is set)
	if !autoConfirm {
		fmt.Print("\nKill this process? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted.")
			return
		}
	}

	// Kill the process(es)
	if err := killProcesses(pids, port); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`killport - Kill processes running on a specific port

Usage:
  killport [options] <port>

Options:
  -y, --yes       Skip confirmation prompt and kill immediately
  -h, --help      Show this help message
  -v, --version   Show version information

Examples:
  killport 3000       Kill the process on port 3000 (with confirmation)
  killport -y 3000    Kill the process on port 3000 (no confirmation)
  killport 8080       Kill the process on port 8080 (with confirmation)`)
}

func findProcessOnPort(port int) (string, []string, error) {
	switch runtime.GOOS {
	case "windows":
		return findProcessWindows(port)
	case "darwin", "linux":
		return findProcessUnix(port)
	default:
		return "", nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func findProcessWindows(port int) (string, []string, error) {
	// Find the PID using netstat
	cmd := exec.Command("cmd", "/c", fmt.Sprintf("netstat -ano | findstr :%d", port))
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return "", nil, nil
	}

	lines := strings.Split(string(output), "\n")
	var pids []string
	var info strings.Builder
	seenPids := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		// Verify the local address contains our port
		localAddr := fields[1]
		if !strings.HasSuffix(localAddr, fmt.Sprintf(":%d", port)) {
			continue
		}

		pid := fields[len(fields)-1]
		if pid == "0" || seenPids[pid] {
			continue
		}

		seenPids[pid] = true
		pids = append(pids, pid)

		// Get process name
		nameCmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %s", pid), "/FO", "CSV", "/NH")
		nameOutput, _ := nameCmd.Output()
		processName := "Unknown"
		if len(nameOutput) > 0 {
			parts := strings.Split(strings.TrimSpace(string(nameOutput)), ",")
			if len(parts) > 0 {
				processName = strings.Trim(parts[0], "\"")
			}
		}

		info.WriteString(fmt.Sprintf("  PID: %s  Name: %s  State: %s\n", pid, processName, fields[3]))
	}

	return info.String(), pids, nil
}

func findProcessUnix(port int) (string, []string, error) {
	// Use lsof to find the process with more details
	cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port))
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return "", nil, nil
	}

	lines := strings.Split(string(output), "\n")
	var pids []string
	var info strings.Builder
	seenPids := make(map[string]bool)

	for i, line := range lines {
		if i == 0 || line == "" { // Skip header
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		pid := fields[1]
		if seenPids[pid] {
			continue
		}

		seenPids[pid] = true
		pids = append(pids, pid)

		processName := fields[0]
		user := ""
		if len(fields) > 2 {
			user = fields[2]
		}

		info.WriteString(fmt.Sprintf("  PID: %s  Name: %s  User: %s\n", pid, processName, user))
	}

	return info.String(), pids, nil
}

func killProcesses(pids []string, port int) error {
	switch runtime.GOOS {
	case "windows":
		return killProcessesWindows(pids, port)
	case "darwin", "linux":
		return killProcessesUnix(pids, port)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func killProcessesWindows(pids []string, port int) error {
	for _, pid := range pids {
		killCmd := exec.Command("taskkill", "/F", "/PID", pid)
		if err := killCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to kill process %s: %v\n", pid, err)
			continue
		}
		fmt.Printf("Killed process %s on port %d\n", pid, port)
	}
	return nil
}

func killProcessesUnix(pids []string, port int) error {
	for _, pid := range pids {
		killCmd := exec.Command("kill", "-9", pid)
		if err := killCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to kill process %s: %v\n", pid, err)
			continue
		}
		fmt.Printf("Killed process %s on port %d\n", pid, port)
	}
	return nil
}
