package raycast

import (
	"os"
	"testing"
)

func TestNewPolygon(t *testing.T) {
	tests := []struct {
		points []Point
		want   error
	}{
		{[]Point{{0, 0}, {1, 0}, {0, 1}}, nil},    // валидный случай
		{[]Point{{0, 0}, {1, 0}}, ErrLess3Points}, // менее 3 точек
		{[]Point{}, ErrLess3Points},               // пустой массив
	}

	for _, tc := range tests {
		_, got := NewPolygon(tc.points)
		if got != tc.want {
			t.Errorf("NewPolygon(%v) = %v; want %v", tc.points, got, tc.want)
		}
	}
}

func TestIsPointInPolygon(t *testing.T) {
	polygon, _ := NewPolygon([]Point{{0, 0}, {5, 0}, {5, 5}, {0, 5}})

	tests := []struct {
		point Point
		want  bool
	}{
		{Point{1, 1}, true},    // внутри
		{Point{5, 1}, true},    // на границе
		{Point{3, 3}, true},    // внутри
		{Point{6, 6}, false},   // снаружи
		{Point{-1, -1}, false}, // снаружи
		{Point{-1, 6}, false},  // снаружи
		{Point{6, -1}, false},  // снаружи
		{Point{0, 0}, true},    // на углу
		{Point{2.5, 0}, true},  // на границе
		{Point{0, 2.5}, true},  // на границе
		{Point{5, 5}, true},    // на углу
	}

	for _, tс := range tests {
		got := IsPointInPolygon(tс.point, polygon)
		if got != tс.want {
			t.Errorf("IsPointInPolygon(%v, polygon) = %v; want %v", tс.point, got, tс.want)
		}
	}
}

func TestLoadPolygonFromFile(t *testing.T) {
	// Создаем временный файл для тестирования
	file, err := os.CreateTemp("", "polygon_test.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(file.Name()) // удаляем файл после теста

	// Записываем тестовые данные в файл
	_, err = file.WriteString("0, 0\n1, 0\n1, 1\n0, 1\n")
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	file.Close() // закрываем файл для чтения

	// Тестируем загрузку полигона из файла
	polygon, err := LoadPolygonFromFile(file.Name())
	if err != nil {
		t.Fatalf("LoadPolygonFromFile() error = %v", err)
	}

	if len(polygon) != 4 {
		t.Errorf("LoadPolygonFromFile() = %v; want 4 segments", len(polygon))
	}
}
