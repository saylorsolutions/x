package iterx

import (
	"strconv"
)

type TableIterator[T any] func(yield func(row int, col int, value T) bool)

// SelectTable will produce a table that may be further filtered and transformed.
//
// Note that this function will panic if rows do now have the same number of columns, as this can violate assumptions in other methods.
func SelectTable[T any](table [][]T) TableIterator[T] {
	iter := TableIterator[T](func(yield func(row int, col int, value T) bool) {
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

func (i TableIterator[T]) Dimensions() (width, height int) {
	var (
		lastRow      = -1
		lastRowWidth int
	)
	i(func(row int, col int, _ T) bool {
		if lastRow == -1 {
			lastRow = row
		}
		if lastRow != row {
			if lastRowWidth != width {
				panic("row with different number of columns found")
			}
			lastRow = row
		}
		lastRowWidth = col + 1
		height = max(row+1, height)
		width = max(lastRowWidth, width)
		return true
	})
	if lastRow != -1 && lastRowWidth != width {
		panic("row with different number of columns found")
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

func (i TableIterator[T]) FilterRows(filter RowFilter[T]) TableIterator[T] {
	return func(yield func(row int, col int, value T) bool) {
		i(func(row int, col int, value T) bool {
			if filter(row, col, value) {
				return yield(row, col, value)
			}
			return true
		})
	}
}

type ColumnFilter[T any] func(colnum int, column T) bool

func SkipColumns[T any](skipped ...int) ColumnFilter[T] {
	skipSet := SliceSet(skipped).Map()
	return func(colnum int, _ T) bool {
		return !skipSet[colnum]
	}
}

func (i TableIterator[T]) FilterColumns(filter ColumnFilter[T]) TableIterator[T] {
	return func(yield func(row int, col int, value T) bool) {
		i(func(row int, col int, value T) bool {
			if filter(col, value) {
				return yield(row, col, value)
			}
			return true
		})
	}
}

func (i TableIterator[T]) SelectColumns(columns ...int) TableIterator[T] {
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

func (i TableIterator[T]) RowOffset(offset int) TableIterator[T] {
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

func (i TableIterator[T]) RowLimit(limit int) TableIterator[T] {
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
func (i TableIterator[T]) RotateTable() MapIter[int, SliceIter[T]] {
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

func TransformRows[T1 any, T2 any](iter TableIterator[T1], transform func(row MapIter[int, T1]) T2) MapIter[int, T2] {
	return func(yield func(int, T2) bool) {
		iter.Rows().ForEach(func(rownum int, row MapIter[int, T1]) bool {
			return yield(rownum, transform(row))
		})
	}
}

func TransformLabeledRows[T1 any, T2 any](iter TableIterator[T1], labels []string, transform func(row MapIter[string, T1]) T2) MapIter[int, T2] {
	return func(yield func(int, T2) bool) {
		iter.LabeledRows(labels).ForEach(func(rownum int, row MapIter[string, T1]) bool {
			return yield(rownum, transform(row))
		})
	}
}

func (i TableIterator[T]) Rows() MapIter[int, MapIter[int, T]] {
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

func (i TableIterator[T]) AppendColumn(colValue func(row MapIter[int, T]) T) TableIterator[T] {
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

func (i TableIterator[T]) LabeledRows(columnLabels []string) MapIter[int, MapIter[string, T]] {
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

func (i TableIterator[T]) Table() [][]T {
	return TransformValues(i.Rows(), func(value MapIter[int, T]) []T {
		return value.Values().Slice()
	}).Values().Slice()
}
