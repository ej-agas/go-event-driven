package api

import (
	"context"
	"sync"
)

type SpreadsheetsAPIMock struct {
	lock sync.Mutex
	Rows map[string][][]string
}

func (s *SpreadsheetsAPIMock) AppendRow(ctx context.Context, spreadsheetName string, row []string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.Rows == nil {
		s.Rows = make(map[string][][]string)
	}

	s.Rows[spreadsheetName] = append(s.Rows[spreadsheetName], row)

	return nil
}
