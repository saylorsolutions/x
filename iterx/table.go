package iterx

import (
	"fmt"
	"strconv"
)

type TableIter[T any] func(yield func(row int, col int, value T) bool)

// SelectTable will produce a TableIter that may be used to further filter and transform the set of values.
//
// Note that this function will panic if rows don't have the same number of columns, as this can violate constraints in other methods.
func SelectTable[T any](table [][]T) TableIter[T] {
	iter := TableIter[T](func(yield func(row int, col int, value T) bool) {
		for rownum, row := range table {
			for colnum, col := range row {
				if !yield(rownum, colnum, col) {
					return
				}
			}
		}
	})
	iter.Dimensions()
	return iter
}

func SelectTableFromRows[T any](rows MapIter[int, MapIter[int, T]]) TableIter[T] {
	return func(yield func(row int, col int, value T) bool) {
		rows.KeyOrder(Sort[int]).ForEach(func(rownum int, row MapIter[int, T]) bool {
			nextRow := true
			row.KeyOrder(Sort[int]).ForEach(func(colnum int, val T) bool {
				nextRow = yield(rownum, colnum, val)
				return nextRow
			})
			return nextRow
		})
	}
}

func (i TableIter[T]) Dimensions() (width, height int) {
	var (
		lastRow      = -1
		lastRowWidth int
	)
	i(func(row int, _ int, _ T) bool {
		if lastRow == -1 {
			lastRow = row
			height = 1
		}
		if lastRow != row {
			if width == 0 {
				width = lastRowWidth
			} else if lastRowWidth != width {
				panic(fmt.Sprintf("row %d has a different number of columns", lastRow))
			}
			lastRow = row
			lastRowWidth = 0
			height++
		}
		lastRowWidth++
		return true
	})
	if width == 0 {
		width = lastRowWidth
	} else if lastRowWidth != width {
		panic("last row has a different number of columns")
	}
	return
}

type RowFilter[T any] func(rownum int, colnum int, col T) bool

func SkipRows[T any](skipped ...int) RowFilter[T] {
	skipSet := SliceSet(skipped).Map()
	return func(rownum int, _ int, col T) bool {
		return !skipSet[rownum]
	}
}

func FilterColumnValue[T any](column int, filter Filter[T]) RowFilter[T] {
	return func(_ int, colnum int, value T) bool {
		if colnum != column {
			return true
		}
		return filter(value)
	}
}

func (i TableIter[T]) FilterRows(filter RowFilter[T]) TableIter[T] {
	return func(yield func(row int, col int, value T) bool) {
		keepIterating := true
		i.Rows().ForEach(func(rownum int, row MapIter[int, T]) bool {
			excludeRow := false
			row.ForEach(func(colnum int, val T) bool {
				if !filter(rownum, colnum, val) {
					excludeRow = true
					return false
				}
				return true
			})
			if !excludeRow {
				row.ForEach(func(colnum int, val T) bool {
					keepIterating = yield(rownum, colnum, val)
					return keepIterating
				})
			}
			return keepIterating
		})
	}
}

func (i TableIter[T]) SkipColumns(excluded ...int) TableIter[T] {
	return func(yield func(row int, col int, value T) bool) {
		exclusionSet := SliceSet(excluded).Map()
		i(func(row int, col int, value T) bool {
			if !exclusionSet[col] {
				return yield(row, col, value)
			}
			return true
		})
	}
}

func (i TableIter[T]) SelectColumns(columns ...int) TableIter[T] {
	colSet := SliceSet(columns).Map()
	return func(yield func(row int, col int, value T) bool) {
		i(func(row int, col int, value T) bool {
			if colSet[col] {
				return yield(row, col, value)
			}
			return true
		})
	}
}

func (i TableIter[T]) RowOffset(offset int) TableIter[T] {
	return func(yield func(row int, col int, value T) bool) {
		var numOffset int
		i(func(row int, col int, value T) bool {
			if numOffset >= offset {
				numOffset++
				return true
			}
			return yield(row, col, value)
		})
	}
}

func (i TableIter[T]) RowLimit(limit int) TableIter[T] {
	return func(yield func(row int, col int, value T) bool) {
		var numSent int
		i(func(row int, col int, value T) bool {
			if numSent >= limit {
				return false
			}
			numSent++
			return yield(row, col, value)
		})
	}
}

// RotateTable will return a map of SliceIter, with a key for each column.
func (i TableIter[T]) RotateTable() MapIter[int, SliceIter[T]] {
	return func(yield func(int, SliceIter[T]) bool) {
		columns := map[int]SliceIter[T]{}
		i(func(row int, col int, value T) bool {
			slice, ok := columns[col]
			if !ok {
				columns[col] = SelectValue(value)
				return true
			}
			columns[col] = slice.AppendValue(value)
			return true
		})
		SelectMap(columns).KeyOrder(Sort[int]).ForEach(yield)
	}
}

func TransformRows[T1 any, T2 any](iter TableIter[T1], transform func(row MapIter[int, T1]) T2) MapIter[int, T2] {
	return func(yield func(int, T2) bool) {
		iter.Rows().ForEach(func(rownum int, row MapIter[int, T1]) bool {
			return yield(rownum, transform(row))
		})
	}
}

func TransformLabeledRows[T1 any, T2 any](iter TableIter[T1], labels []string, transform func(row MapIter[string, T1]) T2) MapIter[int, T2] {
	return func(yield func(int, T2) bool) {
		iter.LabeledRows(labels).ForEach(func(rownum int, row MapIter[string, T1]) bool {
			return yield(rownum, transform(row))
		})
	}
}

func (i TableIter[T]) Rows() MapIter[int, MapIter[int, T]] {
	var (
		rows    = SelectMap[int, MapIter[int, T]](nil)
		curCols MapIter[int, T]
		lastRow = -1
	)
	i(func(row int, col int, value T) bool {
		if row != lastRow {
			if lastRow > -1 {
				rows = rows.AppendEntry(lastRow, curCols)
			}
			curCols = SelectEntry(col, value)
			lastRow = row
			return true
		}
		curCols = curCols.AppendEntry(col, value)
		return true
	})
	if lastRow > -1 {
		rows = rows.AppendEntry(lastRow, curCols)
	}
	return rows
}

func (i TableIter[T]) AppendColumn(colValue func(row MapIter[int, T]) T) TableIter[T] {
	return func(yield func(row int, col int, value T) bool) {
		var (
			lastRow     = -1
			lastCol     = -1
			prevColVals = SelectMap[int, T](nil)
		)
		i(func(row int, col int, value T) bool {
			if lastRow == -1 {
				lastRow = row
			}
			if row != lastRow {
				if !yield(lastRow, lastCol+1, colValue(prevColVals)) {
					return false
				}
				prevColVals = SelectMap[int, T](nil)
			}
			lastRow = row
			lastCol = col
			prevColVals = prevColVals.AppendEntry(col, value)
			return yield(row, col, value)
		})
		if lastRow == -1 {
			yield(0, 0, colValue(prevColVals))
			return
		}
		yield(lastRow, lastCol+1, colValue(prevColVals))
	}
}

func (i TableIter[T]) LabeledRows(columnLabels []string) MapIter[int, MapIter[string, T]] {
	labels := DedupeValues(SliceMap(columnLabels)).Map()
	return TransformValues(i.Rows(), func(rowIter MapIter[int, T]) MapIter[string, T] {
		return TransformKeys(rowIter, func(colnum int) string {
			label, ok := labels[colnum]
			if !ok {
				label = strconv.Itoa(colnum)
			}
			return label
		})
	})
}

func (i TableIter[T]) Table() [][]T {
	return TransformValues(i.Rows(), func(value MapIter[int, T]) []T {
		return value.Values().Slice()
	}).Values().Slice()
}
