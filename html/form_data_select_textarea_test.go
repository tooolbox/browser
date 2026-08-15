package html_test

import (
	"testing"

	"github.com/gost-dom/browser/dom"
	"github.com/gost-dom/browser/html"
	"github.com/gost-dom/browser/internal/testing/htmltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// formFromHTML loads a document and returns its first <form>.
func formFromHTML(t *testing.T, src string) html.HTMLFormElement {
	t.Helper()
	win := htmltest.NewWindowHelper(t, nil)
	win.MustLoadHTML(src)
	el, err := win.Document().QuerySelector("form")
	require.NoError(t, err, "querying for form")
	require.NotNil(t, el, "no form in test HTML")
	form, ok := el.(html.HTMLFormElement)
	require.True(t, ok, "element is not an HTMLFormElement")
	return form
}

// The form data set must include <select> and <textarea> controls, not just
// <input>. https://html.spec.whatwg.org/multipage/form-control-infrastructure.html#constructing-the-form-data-set
func TestFormDataFormIncludesSelectAndTextarea(t *testing.T) {
	form := formFromHTML(t, `<form>
		<input name="username" value="john">
		<select name="colour">
			<option value="red">Red</option>
			<option value="green" selected>Green</option>
			<option value="blue">Blue</option>
		</select>
		<textarea name="comment">hello world</textarea>
	</form>`)

	formData := html.NewFormDataForm(form)

	assert.Equal(t, html.FormDataValue("john"), formData.Get("username"))
	assert.Equal(t, html.FormDataValue("green"), formData.Get("colour"),
		"select submits the option carrying the selected attribute")
	assert.Equal(t, html.FormDataValue("hello world"), formData.Get("comment"),
		"textarea submits its text content")
}

// With no option marked selected, a browser submits the first one.
func TestFormDataFormSelectDefaultsToFirstOption(t *testing.T) {
	form := formFromHTML(t, `<form>
		<select name="colour">
			<option value="red">Red</option>
			<option value="green">Green</option>
		</select>
	</form>`)

	formData := html.NewFormDataForm(form)

	assert.Equal(t, html.FormDataValue("red"), formData.Get("colour"))
}

// An option without a value attribute submits its text content.
func TestFormDataFormSelectOptionWithoutValueUsesText(t *testing.T) {
	form := formFromHTML(t, `<form>
		<select name="colour">
			<option>Red</option>
			<option selected>Green</option>
		</select>
	</form>`)

	formData := html.NewFormDataForm(form)

	assert.Equal(t, html.FormDataValue("Green"), formData.Get("colour"))
}

// A <select multiple> contributes one entry per selected option, in document
// order -- not just the first.
func TestFormDataFormMultipleSelectSubmitsEverySelectedOption(t *testing.T) {
	form := formFromHTML(t, `<form>
		<select name="colour" multiple>
			<option value="red" selected>Red</option>
			<option value="green">Green</option>
			<option value="blue" selected>Blue</option>
		</select>
	</form>`)

	formData := html.NewFormDataForm(form)

	assert.Equal(t,
		[]html.FormDataValue{html.FormDataValue("red"), html.FormDataValue("blue")},
		formData.GetAll("colour"),
		"every selected option is submitted, in document order")
}

// A <select multiple> with nothing selected submits nothing -- the
// first-option fallback applies only to single selects.
func TestFormDataFormMultipleSelectWithNoSelectionSubmitsNothing(t *testing.T) {
	form := formFromHTML(t, `<form>
		<input name="username" value="john">
		<select name="colour" multiple>
			<option value="red">Red</option>
			<option value="green">Green</option>
		</select>
	</form>`)

	formData := html.NewFormDataForm(form)

	assert.False(t, formData.Has("colour"),
		"an untouched multiple select contributes no entry")
	assert.Len(t, formData.Entries, 1, "only the input should be submitted")
}

// A single select with one option selected still yields exactly one entry
// even if later options exist.
func TestFormDataFormSingleSelectSubmitsOneValue(t *testing.T) {
	form := formFromHTML(t, `<form>
		<select name="colour">
			<option value="red" selected>Red</option>
			<option value="green" selected>Green</option>
		</select>
	</form>`)

	formData := html.NewFormDataForm(form)

	assert.Equal(t,
		[]html.FormDataValue{html.FormDataValue("red")},
		formData.GetAll("colour"),
		"a non-multiple select contributes a single value")
}

// A select with no options contributes nothing, rather than an empty value.
func TestFormDataFormSelectWithNoOptionsSubmitsNothing(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
	}{
		{"single", `<form><input name="username" value="john"><select name="colour"></select></form>`},
		{"multiple", `<form><input name="username" value="john"><select name="colour" multiple></select></form>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			formData := html.NewFormDataForm(formFromHTML(t, tc.src))
			assert.False(t, formData.Has("colour"),
				"a select with no options contributes no entry")
			assert.Len(t, formData.Entries, 1)
		})
	}
}

// Unnamed controls are skipped, matching the existing behaviour for inputs.
func TestFormDataFormSkipsUnnamedSelectAndTextarea(t *testing.T) {
	form := formFromHTML(t, `<form>
		<input name="username" value="john">
		<select><option value="red" selected>Red</option></select>
		<textarea>orphan text</textarea>
	</form>`)

	formData := html.NewFormDataForm(form)

	assert.Len(t, formData.Entries, 1, "only the named input should be submitted")
	assert.Equal(t, html.FormDataValue("john"), formData.Get("username"))
}

// Setting the selected attribute after parse -- what a test driving a page
// does -- must change what the form submits.
func TestFormDataFormSelectReflectsAttributeChange(t *testing.T) {
	form := formFromHTML(t, `<form>
		<select name="colour">
			<option value="red">Red</option>
			<option value="green">Green</option>
		</select>
	</form>`)

	opts, err := form.QuerySelectorAll("option")
	require.NoError(t, err)
	green, ok := opts.Item(1).(dom.Element)
	require.True(t, ok)
	green.SetAttribute("selected", "selected")

	formData := html.NewFormDataForm(form)

	assert.Equal(t, html.FormDataValue("green"), formData.Get("colour"))
}
