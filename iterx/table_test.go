package iterx

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestTableMetadata_FilterColumns(t *testing.T) {
	table := [][]int{
		{0, 1, 2},
	}
	tests := map[string]struct {
		excluded int
		expected [][]int
	}{
		"Exclude head": {
			excluded: 0,
			expected: [][]int{{1, 2}},
		},
		"Exclude tail": {
			excluded: 2,
			expected: [][]int{{0, 1}},
		},
		"Exclude middle": {
			excluded: 1,
			expected: [][]int{{0, 2}},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			result := SelectTable(table).SkipColumns(tc.excluded).Table()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTableMetadata_FilterRows(t *testing.T) {
	table := [][]int{
		{0},
		{1},
		{2},
	}
	tests := map[string]struct {
		excluded int
		expected [][]int
	}{
		"Exclude head row": {
			excluded: 0,
			expected: [][]int{{1}, {2}},
		},
		"Exclude tail row": {
			excluded: 2,
			expected: [][]int{{0}, {1}},
		},
		"Exclude middle row": {
			excluded: 1,
			expected: [][]int{{0}, {2}},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			filtered := SelectTable(table).FilterRows(SkipRows[int](tc.excluded))
			result := filtered.Table()
			width, height := filtered.Dimensions()
			assert.Equal(t, tc.expected, result)
			assert.Equal(t, 2, height)
			assert.Equal(t, 1, width)
		})
	}
}

func TestTransformRows(t *testing.T) {
	table := SelectTable([][]string{
		{"A", "54", "blah"},
		{"B", "45", "blah"},
	})
	type transformedRow struct {
		name string
		age  int
		err  error
	}
	transform := func(row MapIter[int, string]) transformedRow {
		cols := row.Values().Slice()
		age, err := strconv.Atoi(cols[1])
		if err != nil {
			return transformedRow{
				err: err,
			}
		}
		return transformedRow{
			name: cols[0],
			age:  age,
		}
	}
	rows := TransformRows(table.SkipColumns(2), transform)
	assert.Equal(t, 2, rows.Count())
	assert.Equal(t, []int{0, 1}, rows.Keys().Slice())
	rows.ForEach(func(row int, val transformedRow) bool {
		assert.NoError(t, val.err)
		switch row {
		case 0:
			assert.Equal(t, "A", val.name)
			assert.Equal(t, 54, val.age)
		case 1:
			assert.Equal(t, "B", val.name)
			assert.Equal(t, 45, val.age)
		default:
			t.Error("Too many rows")
		}
		return true
	})
}

func TestTransformLabeledTable(t *testing.T) {
	table := SelectTable([][]string{
		{"A", "54", "blah"},
		{"B", "45", "blah"},
	})
	type transformedRow struct {
		name string
		age  int
		err  error
	}
	transform := func(row MapIter[string, string]) transformedRow {
		cols := row.Map()
		age, err := strconv.Atoi(cols["age"])
		if err != nil {
			return transformedRow{
				err: err,
			}
		}
		return transformedRow{
			name: cols["name"],
			age:  age,
		}
	}
	rows := TransformLabeledRows(table.SkipColumns(2), []string{"name", "age", "desc"}, transform)
	assert.Equal(t, 2, rows.Count())
	assert.Equal(t, []int{0, 1}, rows.Keys().Slice())
	rows.ForEach(func(row int, val transformedRow) bool {
		assert.NoError(t, val.err)
		switch row {
		case 0:
			assert.Equal(t, "A", val.name)
			assert.Equal(t, 54, val.age)
		case 1:
			assert.Equal(t, "B", val.name)
			assert.Equal(t, 45, val.age)
		default:
			t.Error("Too many rows")
		}
		return true
	})
}

func TestTableIterator_AppendColumn(t *testing.T) {
	table := SelectTable([][]float64{
		{2, 2},
		{3, 3},
		{4, 4},
	})
	result := table.AppendColumn(func(row MapIter[int, float64]) float64 {
		return Sum(row.Values())
	}).AppendColumn(func(row MapIter[int, float64]) float64 {
		return Sum(row.Values())
	}).AppendColumn(func(row MapIter[int, float64]) float64 {
		return Sum(row.Values())
	}).SelectColumns(2, 3, 4).Table()
	expected := [][]float64{
		{4, 8, 16},
		{6, 12, 24},
		{8, 16, 32},
	}
	assert.Equal(t, expected, result)
}

func TestTableIterator_RotateTable(t *testing.T) {
	table := SelectTable([][]string{
		{"A1", "B1"},
		{"A2", "B2"},
	})
	result := TransformSlice(table.RotateTable().Values(), func(val SliceIter[string]) []string {
		return val.Slice()
	}).Slice()
	expected := [][]string{
		{"A1", "A2"},
		{"B1", "B2"},
	}
	assert.Equal(t, expected, result)
}

func TestSelectTable_PanicOnDifferentRowWidths(t *testing.T) {
	t.Run("Expanded first row", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					assert.Contains(t, fmt.Sprintf("%v", r), "1")
				} else {
					t.Error("should have panicked")
				}
			}()
			SelectTable([][]int{
				{0, 1, 2},
				{0, 1},
				{0, 1},
			})
		}()
	})
	t.Run("Expanded middle row", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					assert.Contains(t, fmt.Sprintf("%v", r), "1")
				} else {
					t.Error("should have panicked")
				}
			}()
			SelectTable([][]int{
				{0, 1},
				{0, 1, 2},
				{0, 1},
			})
		}()
	})
	t.Run("Expanded last row", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					assert.Contains(t, fmt.Sprintf("%v", r), "last")
				} else {
					t.Error("should have panicked")
				}
			}()
			SelectTable([][]int{
				{0, 1},
				{0, 1},
				{0, 1, 2},
			})
		}()
	})
	t.Run("Balanced table", func(t *testing.T) {
		assert.NotPanics(t, func() {
			width, height := SelectTable([][]int{
				{0, 1},
				{0, 1},
				{0, 1},
			}).Dimensions()
			assert.Equal(t, 2, width)
			assert.Equal(t, 3, height)
		})
	})
}

func TestFilterColumnValue(t *testing.T) {
	result := SelectTable([][]int{
		{0, 1, 0},
		{1, 0, 0},
		{0, 1, 0},
	}).FilterRows(FilterColumnValue[int](1, NoZeroValues[int]())).
		Table()
	expected := [][]int{
		{0, 1, 0},
		{0, 1, 0},
	}
	assert.Equal(t, expected, result)
}

func TestJoinTable(t *testing.T) {
	a := SelectTable([][]int{
		{0, 1, 2},
		{3, 4, 5},
		{6, 7, 8},
	})
	b := SelectTable([][]int{
		{0, 2, 4},
		{3, 8, 10},
		{6, 14, 16},
	})

	joiner := CompareColumns(0, 2, func(a, b int) bool {
		return a == b-1
	}).Or(CompareColumns(0, 2, func(a, b int) bool {
		return b == a+10
	}))
	result := JoinTable(a, b, joiner)
	expected := [][]int{
		{0, 1, 2, 3, 8, 10},
		{3, 4, 5, 0, 2, 4},
		{6, 7, 8, 6, 14, 16},
	}
	assert.Equal(t, expected, result.Table())
	result(func(row int, col int, value int) bool {
		assert.Equal(t, expected[row][col], value)
		return true
	})
}

func TestTable_OffsetLimit(t *testing.T) {
	table := make([][]int, 1000)
	for i := 0; i < 1000; i++ {
		table[i] = make([]int, 3)
		for j := 0; j < 3; j++ {
			table[i][j] = j + i*3
		}
	}

	tableSel := SelectTable(table)
	t.Run("Offset 100, no limit", func(t *testing.T) {
		assert.Equal(t, 900, tableSel.RowOffset(100).Rows().Count())
	})
	t.Run("Offset 200, no limit", func(t *testing.T) {
		assert.Equal(t, 800, tableSel.RowOffset(200).Rows().Count())
	})
	t.Run("Offset 100, limit 100", func(t *testing.T) {
		assert.Equal(t, 100, tableSel.RowOffset(100).RowLimit(100).Rows().Count())
	})
	t.Run("No offset, limit 100", func(t *testing.T) {
		assert.Equal(t, 100, tableSel.RowLimit(100).Rows().Count())
	})
	t.Run("Offset 200, limit 200", func(t *testing.T) {
		assert.Equal(t, 200, tableSel.RowOffset(200).RowLimit(200).Rows().Count())
	})
	t.Run("Offset 900, limit 200", func(t *testing.T) {
		assert.Equal(t, 100, tableSel.RowOffset(900).RowLimit(200).Rows().Count())
	})
}
