package action

import (
	"io"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Export an action
func Export(db gorp.SqlExecutor, name string, f exportentities.Format, w io.Writer) (int, error) {
	a, err := LoadTypeBuiltInOrDefaultByName(db, name)
	if err != nil {
		return 0, err
	}

	return ExportAction(*a, f, w)
}

// ExportAction export
func ExportAction(a sdk.Action, f exportentities.Format, w io.Writer) (int, error) {
	ea := exportentities.NewAction(a)
	b, err := exportentities.Marshal(ea, f)
	if err != nil {
		return 0, sdk.WithStack(err)
	}

	return w.Write(b)
}
