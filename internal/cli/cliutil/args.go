package cliutil

func IsHelpRequest(args []string) bool {
	if len(args) == 0 {
		return true
	}

	switch args[0] {
	case "help", "-h", "--help", "-help":
		return true
	default:
		return false
	}
}
