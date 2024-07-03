/*
Package stringmetric provides functions to calculate string metrics,
also known as string similarity metrics or string distance functions.

A string metric measures the distance ("inverse similarity") between two text strings
for approximate string matching, comparison, and fuzzy string searching.
*/
package stringmetric

// DLDistance calculates the Damerau-Levenshtein edit distance between two strings.
// It provides the distance between two strings as the minimum number of operations required to change one string into the other.
// The operations include insertions, deletions, substitutions, and transpositions of two adjacent characters.
//
// Ref.: https://en.wikipedia.org/wiki/Damerau%E2%80%93Levenshtein_distance
func DLDistance(sa, sb string) int {
	if sa == sb {
		return 0
	}

	ra := []rune(sa)
	rb := []rune(sb)
	ralen := len(ra)
	rblen := len(rb)
	maxdist := ralen + rblen

	if (ralen == 0) || (rblen == 0) {
		return maxdist
	}

	// initialize alphabet (Σ)
	da := initDLAlphabet(ra, rb, maxdist)

	// initialize distance matrix
	ncols := ralen + 2
	nrows := rblen + 2
	dist := initDLMatrix(ncols, nrows, maxdist)

	// fill the distance matrix
	for i := 2; i < ncols; i++ {
		db := 0

		for j := 2; j < nrows; j++ {
			k := da[rb[j-2]]
			l := db
			tcost := (i - k + j - l - 2) // transposition cost = (i-k-1)+(j-l-1)
			scost := 1                   // substitution cost

			if ra[i-2] == rb[j-2] {
				scost = 0
				db = j
			}

			dist[i][j] = min(
				dist[i-1][j-1]+scost, // substitution
				dist[i][j-1]+1,       // insertion
				dist[i-1][j]+1,       // deletion
				dist[k][l]+tcost,     // transposition
			)
		}

		da[ra[i-2]] = i
	}

	// Example:
	// "a cat" (one transposition)-> "a act" (one insertion)-> "a abct"
	//
	//	              a  n     a  c  t
	//	        0  1  2  3  4  5  6  7
	//	     +-------------------------+
	//	   0 | 11 11 11 11 11 11 11 11 |
	//	   1 | 11  0  1  2  3  4  5  6 |
	//	a  2 | 11  1  0  1  2  3  4  5 |
	//	   3 | 11  2  1  0  1  2  3  4 |
	//	c  4 | 11  3  2  1  1  2  2  3 |
	//	a  5 | 11  4  3  2  1  2  2  3 |
	//	t  6 | 11  5  4  3  2  2  3  2 |
	//	     +-------------------------+

	return dist[ncols-1][nrows-1] // bottom right value
}

// initDLAlphabet initialize the alphabet (Σ) for the Damerau-Levenshtein distance calculation.
func initDLAlphabet(ra, rb []rune, maxdist int) map[rune]int {
	da := make(map[rune]int, maxdist)

	for _, c := range ra {
		da[c] = 0
	}

	for _, c := range rb {
		da[c] = 0
	}

	return da
}

// initDLMatrix create and initialize the Damerau-Levenshtein distance matrix
// by populating the first two rows and columns.
//
// Example:
//
//	              a  n     a  c  t
//	        0  1  2  3  4  5  6  7
//	     +-------------------------+
//	   0 | 11 11 11 11 11 11 11 11 |
//	   1 | 11  0  1  2  3  4  5  6 |
//	a  2 | 11  1  0  0  0  0  0  0 |
//	   3 | 11  2  0  0  0  0  0  0 |
//	c  4 | 11  3  0  0  0  0  0  0 |
//	a  5 | 11  4  0  0  0  0  0  0 |
//	t  6 | 11  5  0  0  0  0  0  0 |
//	     +-------------------------+
func initDLMatrix(ncols, nrows, maxdist int) [][]int {
	dist := make([][]int, ncols)

	for i := range ncols {
		dist[i] = make([]int, nrows)
		dist[i][0] = maxdist
		dist[i][1] = i - 1
	}

	dist[0][1] = maxdist

	for i := 2; i < nrows; i++ {
		dist[0][i] = maxdist
		dist[1][i] = i - 1
	}

	return dist
}
