package configfile

func Changed(before, after string) bool {
	return before != after
}
