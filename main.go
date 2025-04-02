package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Scrimzay/basicshell/curllogic"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	for {
		// get current working directory
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Println(os.Stderr, "Error getting current directory:", err)
			cwd = "unknown" // Fallback in case of error
		}

		// print instead of println for formatting
		fmt.Printf("BS %s> ", cwd)
		// read keyboard input
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		// remove the newline char
		// this fixes the issue with: input = strings.TrimSuffix(input, "\n")
		// by ensuring not only is the newline removed but ALL whitespace
		// windows new line is /r/n, unix is /n, so the above is fine for unix
		// but windows, the below fixes the issue
		input = strings.TrimSpace(input)

		// skip an empty input
		if input == "" {
			continue
		}

		// handle the execution of the input
		if err = execInput(input); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

// returned when 'cd' was called without a second arg
var ErrNoPath = errors.New("path required")
var ErrNoArg = errors.New("argument required")
var ErrNonCmd = errors.New("not a valid command")
var ErrInvalidSyntax = errors.New("invalid syntax")

// array of available inputs
var commandsArray = []string{
	"cd",
	"cd..",
	"dir",
	"echo",
	"type",
	"del",
	"mkdir",
	"copy",
	"curl",
	"ipconfig",
	"systeminfo",
	"cls",
	"help",
	"exit",
}

func execInput(input string) error {
	// check for piping first
	if strings.Contains(input, "|") {
		return handlePiping(input)
	}

	var cmdArgs, filename string
	// check for redirection
	if strings.Contains(input, ">") {
		// splits the command and filename
		parts := strings.SplitN(input, ">", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
			return ErrInvalidSyntax
		}
		cmdArgs = strings.TrimSpace(parts[0])
		filename = strings.TrimSpace(parts[1])
	} else {
		cmdArgs = input
	}

	// split the input separate the command and the arguments
	args := strings.Split(cmdArgs, " ")

	// handle 'cd..' (without the space, since windows has cd ..)
	// works outside of the switch "cd" case cause you modify the args slice
	// before the switch evaluates the commands
	if len(args) == 1 && cmdArgs == "cd.." {
		args = []string{"cd", ".."} // convert cd.. to cd ..
	}

	// variable specifically just for checking if valid command
	command := args[0]
	isValidCommand := false
	for _, cmd := range commandsArray {
		if command == cmd {
			isValidCommand = true
			break
		}
	}
	if !isValidCommand {
		return ErrNonCmd
	}

	// check for bill tin commands
	switch args[0] {
	case "cd":
		// 'cd' to come home with empty path not yet supported
		if len(args) < 2 {
			return ErrNoPath
		}
		// change the directory and return the error
		return os.Chdir(filepath.FromSlash(args[1]))

	case "echo":
		var output string
		if len(args) < 2 {
			return ErrNoArg
		} else {
			// join the arguments after "echo" and print them
			output = strings.Join(args[1:], " ")
		}

		if filename != "" {
			// Redirect to file
			err := os.WriteFile(filename, []byte(output + "\n"), 0644)
			if err != nil {
				return fmt.Errorf("Failed to write to file: %v", err)
			}
			return nil
		}

		// normal echo to stdout
		fmt.Println(output)
		return nil

	case "type":
		if len(args) < 2 {
			return ErrNoArg
		}
		content, err := os.ReadFile(args[1])
		if err != nil {
			return fmt.Errorf("Failed to read file: %v", err)
		}
		if filename != ""{
			err := os.WriteFile(filename, content, 0644)
			if err != nil {
				return fmt.Errorf("Failed to write to file: %v", err)
			}
			return nil
		}
		fmt.Print(string(content))
		return nil

	case "del":
		if len(args) < 2 {
			return ErrNoArg
		}
		err := os.Remove(args[1])
		if err != nil {
			return fmt.Errorf("Failed to delete file: %v", err)
		}
		return nil

	case "mkdir":
		if len(args) < 2 {
			return ErrNoArg
		}
		err := os.Mkdir(args[1], 0755)
		if err != nil {
			return fmt.Errorf("Failed to create directory: %v", err)
		}
		return nil

	case "copy":
		if len(args) < 3 {
			return ErrNoArg
		}
		content, err := os.ReadFile(args[1])
		if err != nil {
			return fmt.Errorf("Failed to read source file: %v", err)
		}
		err = os.WriteFile(args[2], content, 0644)
		if err != nil {
			return fmt.Errorf("Failed to write to destination file: %v", err)
		}
		if filename != "" {
			err := os.WriteFile(filename, content, 0644)
			if err != nil {
				return fmt.Errorf("Failed to write to redirected file: %v", err)
			}
		}
		return nil

	case "curl":
		curlArgs, err := curllogic.ParseCurlArgs(args, nil)
		if err != nil {
			return err
		}
		if filename != "" {
			// redirect output to file instead of stdout
			outFile, err := os.Create(filename)
			if err != nil {
				return fmt.Errorf("Failed to create file: %v", err)
			}
			defer outFile.Close()
			oldStdout := os.Stdout
			os.Stdout = outFile
			defer func() { os.Stdout = oldStdout }()
		}
		return curllogic.ExecuteCurl(curlArgs, os.Stdout)

	case "ipconfig":
		output, err := exec.Command("cmd.exe", "/C", "ipconfig").Output()
		if err != nil {
			return fmt.Errorf("Failed to run ipconfig command: %v", err)
		}
		if filename != "" {
			err = os.WriteFile(filename, output, 0644)
			if err != nil {
				return fmt.Errorf("Failed to write ipconfig output to file: %v", err)
			}
			return nil
		}
		fmt.Println(string(output))
		return nil

	case "systeminfo":
		output, err := exec.Command("cmd.exe", "/C", "systeminfo").Output()
		if err != nil {
			return fmt.Errorf("Failed to run systeminfo command: %v", err)
		}
		if filename != "" {
			err = os.WriteFile(filename, output, 0644)
			if err != nil {
				return fmt.Errorf("Failed to write systeminfo output to file: %v", err)
			}
			return nil
		}
		fmt.Println(string(output))
		return nil

	case "cls":
		fmt.Print("\033[2J\033[1;1H") // ANSI: Clear screen and move cursor to top-left
		return nil

	case "help":
		fmt.Print("\n")
		for _, list := range commandsArray {
			// fmt.Print(list + "\n")
			switch list {
			case "cd":
				fmt.Println(list + " - Change a directory by going deeper/forward")

			case "cd..":
				fmt.Println(list + " - Change directory by going higher/backwards")

			case "dir":
				fmt.Println(list + " - Show everything inside of the current directory")
			
			case "echo":
				fmt.Println(list + " - Print out an argument")
			
			case "type":
				fmt.Println(list + " - Print out the contents of a specified text file")
			
			case "del":
				fmt.Println(list + " - Delete a specified file within the current directory")
			
			case "mkdir":
				fmt.Println(list + " - Make a new directory within the current directory")
			
			case "copy":
				fmt.Println(list + " - Copy the contents of a source file to a destination file")
			
			case "curl":
				fmt.Println(list + " - Perform a GET request to a specified link with optionally added -X and -H arguments")
			
			case "ipconfig":
				fmt.Println(list + " - Show the network information of the current machine")

			case "systeminfo":
				fmt.Println(list + " - Show more detailed information about the current machine")

			case "cls":
				fmt.Println(list + " - Clears the terminal screen")
			
			case "help":
				fmt.Println(list + " - Show the current list of available commands")
			
			case "exit":
				fmt.Println(list + " - End the current shell session\n")
			
			}
		}

	case "exit":
		os.Exit(0)

	default:
		// prepare the command to execute
		// "/C" tells cmd.exe to execute the command and then terminate
		// also using cmd.exe cause windows crutch
		cmd := exec.Command("cmd.exe", "/C", input)

		if filename != "" {
			// redirect output to file
			file, err := os.Create(filename)
			if err != nil {
				return fmt.Errorf("Failed to create file: %v", err)
			}
			defer file.Close()
			cmd.Stdout = file
			cmd.Stderr = os.Stderr
		} else {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		cmd.Stdin = os.Stdin

		// execute the command return the error
		return cmd.Run()
	}

	return nil
}

func handlePiping(input string) error {
    parts := strings.SplitN(input, "|", 2)
    if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
        return ErrInvalidSyntax
    }
    leftCmd := strings.TrimSpace(parts[0])
    rightInput := strings.TrimSpace(parts[1])

    // Check for redirection in the right side
    var rightCmd, filename string
    if strings.Contains(rightInput, ">") {
        rightParts := strings.SplitN(rightInput, ">", 2)
        if len(rightParts) != 2 || strings.TrimSpace(rightParts[1]) == "" {
            return ErrInvalidSyntax
        }
        rightCmd = strings.TrimSpace(rightParts[0])
        filename = strings.TrimSpace(rightParts[1])
    } else {
        rightCmd = rightInput
    }

    // Split and validate left command
    leftArgs := strings.Split(leftCmd, " ")
    if len(leftArgs) == 1 && leftCmd == "cd.." {
        leftArgs = []string{"cd", ".."}
    }
    if !isValidCommand(leftArgs[0]) {
        return ErrNonCmd
    }

    // Execute left command and capture output
    var output []byte
    var err error
    switch leftArgs[0] {
    case "cd":
        if len(leftArgs) < 2 {
            return ErrNoPath
        }
        err = os.Chdir(filepath.FromSlash(leftArgs[1]))
        if err != nil {
            return err
        }
        output = []byte("") // cd doesn’t produce output

    case "echo":
        if len(leftArgs) < 2 {
            return ErrNoArg
        }
        output = []byte(strings.Join(leftArgs[1:], " ") + "\n")

	case "type":
		if len(leftArgs) < 2 {
			return ErrNoArg
		}
		output, err = os.ReadFile(leftArgs[1])
		if err != nil {
			return fmt.Errorf("Failed to read file: %v", err)
		}

	case "mkdir":
		if len(leftArgs) < 2 {
			return ErrNoArg
		}
		err := os.Mkdir(leftArgs[1], 0755)
		if err != nil {
			return fmt.Errorf("Failed to create directory: %v", err)
		}
		output = []byte("")

	case "copy":
		if len(leftArgs) < 3 {
			return fmt.Errorf("Copy requires source and destination")
		}
		output, err = os.ReadFile(leftArgs[1])
		if err != nil {
			return fmt.Errorf("Failed to read source file: %v", err)
		}
		err = os.WriteFile(leftArgs[2], output, 0644)
		if err != nil {
			return fmt.Errorf("Failed to write to destination file: %v", err)
		}

	case "curl":
		curlArgs, err := curllogic.ParseCurlArgs(leftArgs, nil)
		if err != nil {
			return err
		}
		// capture output for piping
		var buf bytes.Buffer
		err = curllogic.ExecuteCurl(curlArgs, &buf)
		if err != nil {
			return err
		}
		output = buf.Bytes()

    case "dir":
        cmd := exec.Command("cmd.exe", "/C", leftCmd)
        output, err = cmd.Output()
        if err != nil {
            return err
        }
    }

    // Execute right command with piped input
    rightArgs := strings.Split(rightCmd, " ")
    if !isValidCommand(rightArgs[0]) {
        return ErrNonCmd
    }

    switch rightArgs[0] {
	case "cd":
        if len(rightArgs) < 2 {
            return ErrNoPath
        }
        return os.Chdir(filepath.FromSlash(rightArgs[1]))

    case "echo": // Simple case: echo just outputs its input
        if filename != "" {
            err := os.WriteFile(filename, output, 0644)
            if err != nil {
                return fmt.Errorf("Failed to write to file: %v", err)
            }
            return nil
        }
        fmt.Print(string(output))
        return nil

	case "type":
		if len(rightArgs) < 2 {
			return ErrNoArg
		}
		err := os.WriteFile(rightArgs[1], output, 0644)
		if err != nil {
			return fmt.Errorf("Failed to write to file: %v", err)
		}
		if filename != "" {
			err := os.WriteFile(filename, output, 0644)
			if err != nil {
				return fmt.Errorf("Failed to write to file: %v", err)
			}
		}
		return nil

	case "del":
		// use piped input as the filename to delete
		filenameToDelete := strings.TrimSpace(string(output))
		if filenameToDelete == "" {
			return ErrNoArg
		}
		err := os.Remove(filenameToDelete)
		if err != nil {
			return fmt.Errorf("Failed to delete file: %v", err)
		}
		return nil

	case "mkdir":
		filenameToCreate := strings.TrimSpace(string(output))
		if filenameToCreate == "" {
			return ErrNoArg
		}
		err := os.Mkdir(filenameToCreate, 0755)
		if err != nil {
			return fmt.Errorf("Failed to create directory: %v", err)
		}
		return nil

	case "copy":
		if len(rightArgs) < 2 {
			return fmt.Errorf("Copy requires destination with piped input")
		}
		err := os.WriteFile(rightArgs[1], output, 0644)
		if err != nil {
			return fmt.Errorf("Failed to write to destination file: %v", err)
		}
		if filename != "" {
			err := os.WriteFile(filename, output, 0644)
			if err != nil {
				return fmt.Errorf("Failed to write to redirected file: %v", err)
			}
		}
		return nil

	case "curl":
		curlArgs, err := curllogic.ParseCurlArgs(rightArgs, output)
        if err != nil {
            return err
        }
        if filename != "" {
            outFile, err := os.Create(filename)
            if err != nil {
                return fmt.Errorf("Failed to create file: %v", err)
            }
            defer outFile.Close()
            oldStdout := os.Stdout
            os.Stdout = outFile
            defer func() { os.Stdout = oldStdout }()
        }
        return curllogic.ExecuteCurl(curlArgs, os.Stdout)

    case "dir":
        cmd := exec.Command("cmd.exe", "/C", rightCmd)
        cmd.Stdin = bytes.NewReader(output)
        if filename != "" {
            file, err := os.Create(filename)
            if err != nil {
                return fmt.Errorf("Failed to create file: %v", err)
            }
            defer file.Close()
            cmd.Stdout = file
        } else {
            cmd.Stdout = os.Stdout
        }
        cmd.Stderr = os.Stderr
        return cmd.Run()
	}

	return nil
}

func isValidCommand(cmd string) bool {
    for _, validCmd := range commandsArray {
        if cmd == validCmd {
            return true
        }
    }
    return false
}