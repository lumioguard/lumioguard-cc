package comparison

// renameMap finds files moved without edits: an old path whose hash now exists at
// exactly one new path. Ambiguous matches are skipped, never guessed.
func renameMap(current, previous map[string]string) map[string]string {
	byHash := make(map[string][]string, len(current))
	for file, hash := range current {
		byHash[hash] = append(byHash[hash], file)
	}
	renames := make(map[string]string)
	for oldFile, hash := range previous {
		if current[oldFile] == hash {
			continue
		}
		if candidates := byHash[hash]; len(candidates) == 1 {
			renames[oldFile] = candidates[0]
		}
	}
	return renames
}
