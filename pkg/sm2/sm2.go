// Package sm2 реализует алгоритм интервального повторения SuperMemo SM-2.
//
// Шкала оценки (quality 0–5):
//
//	5 — идеальный ответ
//	4 — правильный ответ после небольшой паузы
//	3 — правильный ответ с трудом
//	2 — неправильный ответ, но правильный было легко вспомнить
//	1 — неправильный ответ, правильный трудно вспомнить
//	0 — полный провал
package sm2

import (
	"math"
	"time"
)

const (
	MinEaseFactor   = 1.3
	InitEaseFactor  = 2.5
	MasteredMinReps = 3 // минимум повторений для статуса "mastered"
)

// Result содержит обновлённые параметры карточки после одного повторения.
type Result struct {
	EaseFactor   float64
	IntervalDays int
	Repetitions  int
	NextReviewAt time.Time
	Status       string // "learning" | "review" | "mastered"
}

// Review применяет алгоритм SM-2 и возвращает новые параметры.
//
// Параметры:
//   - easeFactor  — текущий коэффициент лёгкости (обычно начинается с 2.5)
//   - intervalDays — текущий интервал в днях
//   - repetitions  — кол-во успешных ответов подряд
//   - quality      — оценка ответа (0–5)
func Review(easeFactor float64, intervalDays, repetitions, quality int) Result {
	if easeFactor < MinEaseFactor {
		easeFactor = MinEaseFactor
	}

	if quality >= 3 {
		// Правильный ответ: увеличиваем интервал
		var newInterval int
		switch repetitions {
		case 0:
			newInterval = 1
		case 1:
			newInterval = 6
		default:
			newInterval = int(math.Round(float64(intervalDays) * easeFactor))
		}

		// Корректируем коэффициент лёгкости
		delta := 0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02)
		newEase := easeFactor + delta
		if newEase < MinEaseFactor {
			newEase = MinEaseFactor
		}

		newReps := repetitions + 1
		status := statusFromReps(newReps, newInterval)

		return Result{
			EaseFactor:   newEase,
			IntervalDays: newInterval,
			Repetitions:  newReps,
			NextReviewAt: time.Now().AddDate(0, 0, newInterval),
			Status:       status,
		}
	}

	// Неправильный ответ: сбрасываем повторения, повторяем скоро
	return Result{
		EaseFactor:   easeFactor, // ease factor не меняем при ошибке
		IntervalDays: 1,
		Repetitions:  0,
		NextReviewAt: time.Now().Add(10 * time.Minute),
		Status:       "learning",
	}
}

// IsCorrect возвращает true, если ответ считается правильным (quality >= 3).
func IsCorrect(quality int) bool {
	return quality >= 3
}

func statusFromReps(repetitions, intervalDays int) string {
	if repetitions >= MasteredMinReps && intervalDays >= 21 {
		return "mastered"
	}
	if intervalDays >= 7 {
		return "review"
	}
	return "learning"
}
