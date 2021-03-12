package rotate

import "testing"

func Test(t *testing.T) {
	// no compress
	// rotate when reach size limit
	// clean when reach backups limit
	// restart program

	// compress
	// rotate when reach size limit and gzip it
	// after rotate merge gzip if multiple adjacent gzip not reach the size limit
	// after merge gzip rename the first of the merging bundle to the last of the bundle
	// clean when reach backups limit
	// restart program

	// delete backups after the last log of a backup beyond max age
}