package bm25

import "sync"

// Параметры насыщения частоты термина в классической формуле BM25.
// k1 ~ 1.2-2.0 контролирует, насколько быстро "выходит на плато"
// вклад повторного вхождения термина в документ; b ~ 0.75 — насколько
// сильно штрафуются длинные документы. Значения по умолчанию из
// оригинальной статьи Robertson/Sparck Jones, подходят почти для
// любого текстового корпуса без специальной настройки.
const (
	K1 = 1.2
	B  = 0.75
)

// AvgDocLength — скользящая оценка средней длины документа (в
// токенах после Tokenize), нужная для нормализации по длине в
// TF-части формулы. В отличие от IDF (который целиком считает сам
// Qdrant через modifier: idf на sparse-поле коллекции — см.
// комментарий в SparseVector ниже), эта величина не обязана быть
// идеально точной: она лишь мягко корректирует вес термина в
// зависимости от того, длиннее или короче документ среднего, и
// ошибка в ней на десятки процентов не ломает ранжирование
// драматически. Поэтому можно спокойно стартовать с грубой оценки
// и уточнять её по ходу индексации, не пересчитывая уже
// проиндексированные документы.
type AvgDocLength struct {
	mu        sync.Mutex
	totalLen  uint64
	totalDocs uint64
}

// NewAvgDocLength создаёт трекер с "затравкой" — initialEstimate
// (примерная ожидаемая длина чанка в токенах) считается как будто
// уже была подтверждена на seedWeight документах. Это не даёт
// среднему резко скакать на первых нескольких реальных чанках.
func NewAvgDocLength(initialEstimate float64) *AvgDocLength {
	const seedWeight = 20
	return &AvgDocLength{
		totalLen:  uint64(initialEstimate * seedWeight),
		totalDocs: seedWeight,
	}
}

func (a *AvgDocLength) Value() float64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.totalDocs == 0 {
		return 1
	}
	return float64(a.totalLen) / float64(a.totalDocs)
}

func (a *AvgDocLength) observe(docLen int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.totalLen += uint64(docLen)
	a.totalDocs++
}

// SparseVector считает ТОЛЬКО TF-часть формулы BM25 для одного чанка
// (насыщение частоты термина с учётом длины ЭТОГО документа) и
// возвращает готовые indices/values для qdrant.NewVectorSparse.
//
// IDF (обратная частота документа — та самая часть, ради которой
// раньше казалось, что нужно сначала обработать весь корпус) сюда
// сознательно НЕ включается. Если создать sparse-поле коллекции в
// Qdrant с modifier: idf, Qdrant сам применит IDF-взвешивание на
// момент поиска, посчитанное по текущему состоянию коллекции — и
// пересчитывает его динамически по мере добавления новых документов.
// Это и позволяет индексировать чанки по одному, по ходу парсинга,
// не дожидаясь, пока распарсится всё законодательство целиком.
//
// Использовать при ИНДЕКСАЦИИ (не при поиске) — здесь новые слова
// добавляются в словарь через vocab.IDs. Для текста запроса
// используйте QuerySparseVector.
func SparseVector(vocab *Vocabulary, avgDL *AvgDocLength, text string) (indices []uint32, values []float32) {
	tokens := Tokenize(text)
	if len(tokens) == 0 {
		return nil, nil
	}

	termFreq := make(map[string]int, len(tokens))
	for _, t := range tokens {
		termFreq[t]++
	}

	docLen := float64(len(tokens))
	avgdl := avgDL.Value()
	avgDL.observe(len(tokens))

	terms := make([]string, 0, len(termFreq))
	for t := range termFreq {
		terms = append(terms, t)
	}
	ids := vocab.IDs(terms)

	indices = make([]uint32, len(terms))
	values = make([]float32, len(terms))
	for i, t := range terms {
		tf := float64(termFreq[t])
		tfComponent := tf * (K1 + 1) / (tf + K1*(1-B+B*docLen/avgdl))
		indices[i] = ids[i]
		values[i] = float32(tfComponent)
	}
	return indices, values
}

// QuerySparseVector считает sparse-вектор для ТЕКСТА ПОИСКОВОГО
// ЗАПРОСА. В отличие от SparseVector:
//   - использует Lookup, а не IDs — слово, которого нет ни в одном
//     проиндексированном чанке, всё равно ни с чем не совпадёт при
//     dot-product поиске, поэтому не тратим на него новый id;
//   - не применяет насыщение по длине документа — запрос обычно
//     короткий, и здесь важнее просто отметить, какие термины в нём
//     есть; используется чистая частота термина в запросе.
//
// IDF Qdrant применит сам, так же как и при индексации.
func QuerySparseVector(vocab *Vocabulary, text string) (indices []uint32, values []float32) {
	tokens := Tokenize(text)
	if len(tokens) == 0 {
		return nil, nil
	}

	termFreq := make(map[string]int, len(tokens))
	for _, t := range tokens {
		termFreq[t]++
	}

	indices = make([]uint32, 0, len(termFreq))
	values = make([]float32, 0, len(termFreq))
	for t, tf := range termFreq {
		id, ok := vocab.Lookup(t)
		if !ok {
			continue
		}
		indices = append(indices, id)
		values = append(values, float32(tf))
	}
	return indices, values
}
