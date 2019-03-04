package cursor

import (
	"context"
	"github.com/logrange/logrange/pkg/lql"
	"github.com/logrange/logrange/pkg/model"
	"github.com/logrange/logrange/pkg/model/tag"
	"github.com/logrange/range/pkg/records"
	"io"
	"testing"
)

type testLogEventsWrapper struct {
	les []model.LogEvent
	idx int
}

func newTestLogEventsWrapper(les []model.LogEvent) *testLogEventsWrapper {
	return &testLogEventsWrapper{les, 0}
}

func (tle *testLogEventsWrapper) Next(ctx context.Context) {
	if tle.idx < len(tle.les) {
		tle.idx++
	}
}

func (tle *testLogEventsWrapper) Get(ctx context.Context) (records.Record, error) {
	if tle.idx < len(tle.les) {
		buf := make([]byte, tle.les[tle.idx].WritableSize())
		tle.les[tle.idx].Marshal(buf)
		return buf, nil
	}
	return nil, io.EOF
}

func TestFilter(t *testing.T) {
	les := []model.LogEvent{{1, "asdfasdf"}, {2, "as2df"}, {3, "asd3f"}, {4, "jjjj"},
		{5, "jjjjee"}}
	lew := (&model.LogEventIterator{}).Wrap(tag.Line("aaa=bbb"), newTestLogEventsWrapper(les))

	exp, err := lql.ParseExpr("ts = 4 OR msg contains 'asdf'")
	if err != nil {
		t.Fatal("unexpected err=", err)
	}
	fit, err := newFIterator(lew, exp)
	if err != nil {
		t.Fatal("unexpected err=", err)
	}

	le, _, err := fit.Get(nil)
	if le != les[0] {
		t.Fatal("Expected ", les[0], " but received ", le)
	}

	fit.Next(nil)
	le, _, err = fit.Get(nil)
	if le != les[3] {
		t.Fatal("Expected ", les[3], " but received ", le)
	}

	fit.Next(nil)
	le, _, err = fit.Get(nil)
	if err != io.EOF {
		t.Fatal("Expected err==io.EOR, but err=", err, " le=", le)
	}
}