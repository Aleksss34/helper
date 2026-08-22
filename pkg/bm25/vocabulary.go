package bm25

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/kljensen/snowball/russian"
)

// Vocabulary — персистентный словарь "токен -> числовой id" для BM25
// sparse-векторов. Растёт по ходу парсинга: новое слово получает
// следующий свободный id в момент первой встречи. Уже выданные id
// НИКОГДА не меняются и не переиспользуются — иначе уже сохранённые
// в Qdrant sparse-векторы "поплывут", начав ссылаться на чужие
// термины после перезапуска сервиса.
//
// Именно поэтому не нужно сначала "закрывать" весь корпус: словарь
// можно докручивать бесконечно, при появлении новой статьи законов
// просто появятся новые id, а старые чанки останутся валидны.
type Vocabulary struct {
	mu      sync.Mutex
	path    string
	term2id map[string]uint32
	nextID  uint32
}

// LoadVocabulary загружает словарь с диска. Если файла ещё нет —
// это нормальная ситуация (первый запуск), возвращается пустой
// словарь, который будет наполняться по ходу работы.
func LoadVocabulary(path string) (*Vocabulary, error) {
	v := &Vocabulary{
		path:    path,
		term2id: make(map[string]uint32),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return v, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &v.term2id); err != nil {
		return nil, err
	}
	for _, id := range v.term2id {
		if id >= v.nextID {
			v.nextID = id + 1
		}
	}
	return v, nil
}

// Save атомарно сохраняет текущее состояние словаря на диск (запись
// во временный файл + rename, чтобы не оставить битый JSON при
// падении процесса посередине записи). Вызывайте периодически по
// ходу парсинга (например, раз в N статей) и обязательно в конце.
func (v *Vocabulary) Save() error {
	v.mu.Lock()
	data, err := json.Marshal(v.term2id)
	v.mu.Unlock()
	if err != nil {
		return err
	}

	tmp := v.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, v.path)
}

// Size возвращает текущий размер словаря (для логов/метрик).
func (v *Vocabulary) Size() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.term2id)
}

// IDs возвращает id для каждого термина, заводя новую запись в
// словаре, если термин встретился впервые. Использовать при
// ИНДЕКСАЦИИ документа (чанка) — там мы хотим, чтобы новые слова
// становились частью словаря.
func (v *Vocabulary) IDs(terms []string) []uint32 {
	v.mu.Lock()
	defer v.mu.Unlock()

	ids := make([]uint32, len(terms))
	for i, t := range terms {
		id, ok := v.term2id[t]
		if !ok {
			id = v.nextID
			v.term2id[t] = id
			v.nextID++
		}
		ids[i] = id
	}
	return ids
}

// Lookup возвращает id термина, ЕСЛИ он уже есть в словаре, без
// добавления новых записей. Использовать при обработке ПОИСКОВОГО
// запроса: слово, которого нет ни в одном проиндексированном чанке,
// всё равно ни с чем не совпадёт, поэтому нет смысла тратить на него
// новый id.
func (v *Vocabulary) Lookup(term string) (uint32, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	id, ok := v.term2id[term]
	return id, ok
}

// wordRe вырезает "слова" как последовательности юникод-букв/цифр —
// работает как для кириллицы, так и для латиницы/цифр в тексте.
var wordRe = regexp.MustCompile(`[\p{L}\p{N}]+`)

// stopWords — частотные русские служебные слова. Без фильтрации они
// заняли бы огромную долю ненулевых элементов почти в каждом
// sparse-векторе и только "размывали" бы BM25-скор, не неся никакого
// смысла для поиска по законодательству. Список не претендует на
// лингвистическую полноту — это осознанно урезанный набор самых
// частых предлогов/союзов/частиц.
var stopWords = map[string]bool{
	"и": true, "в": true, "во": true, "не": true, "что": true, "он": true,
	"на": true, "я": true, "с": true, "со": true, "как": true, "а": true,
	"то": true, "все": true, "она": true, "так": true, "его": true, "но": true,
	"да": true, "ты": true, "к": true, "у": true, "же": true, "вы": true,
	"за": true, "бы": true, "по": true, "только": true, "ее": true, "мне": true,
	"было": true, "вот": true, "от": true, "меня": true, "еще": true, "нет": true,
	"о": true, "из": true, "ему": true, "теперь": true, "когда": true, "даже": true,
	"ну": true, "вдруг": true, "ли": true, "если": true, "уже": true, "или": true,
	"ни": true, "быть": true, "был": true, "него": true, "до": true, "вас": true,
	"нибудь": true, "опять": true, "уж": true, "вам": true, "ведь": true,
	"там": true, "потом": true, "себя": true, "ничего": true, "ей": true, "может": true,
	"они": true, "тут": true, "где": true, "есть": true, "надо": true, "ней": true,
	"для": true, "мы": true, "тебя": true, "их": true, "чем": true, "была": true,
	"сам": true, "чтоб": true, "без": true, "будто": true, "чего": true, "раз": true,
	"тоже": true, "себе": true, "под": true, "будет": true, "тогда": true,
	"кто": true, "этот": true, "того": true, "потому": true, "этого": true, "какой": true,
	"совсем": true, "ним": true, "здесь": true, "этом": true, "один": true, "почти": true,
	"мой": true, "тем": true, "чтобы": true, "нее": true, "при": true,
}

// Tokenize приводит текст к нижнему регистру, разбивает на слова
// (юникод-буквы/цифры), отбрасывает стоп-слова и однобуквенные
// токены (инициалы, союзы-огрызки и т.п.).
//
// Без лемматизации/стемминга: "полномочия", "полномочий",
// "полномочиями" останутся разными токенами. Для точного лексического
// поиска (что и даёт BM25 в дополнение к семантическому dense-поиску)
// это осознанный компромисс — стемминг иногда объединяет разные по
// смыслу слова и вносит собственный класс ошибок. Если заметите, что
// словоформы мешают находить нужные статьи, можно добавить
// github.com/kljensen/snowball (russian stemmer) прямо здесь, в одном
// месте.
func Tokenize(text string) []string {
	text = strings.ToLower(text)
	raw := wordRe.FindAllString(text, -1)

	tokens := make([]string, 0, len(raw))

	for _, w := range raw {
		if stopWords[w] {
			continue
		}

		// Числа сохраняем независимо от длины.
		if isNumber(w) {
			tokens = append(tokens, w)
			continue
		}

		// Для обычных слов отбрасываем однобуквенные токены.
		if len([]rune(w)) < 2 {
			continue
		}

		stemmed := russian.Stem(w, false)
		tokens = append(tokens, stemmed)
	}
	fmt.Println(tokens)
	return tokens
}

func isNumber(s string) bool {
	for _, r := range s {
		if !unicode.IsNumber(r) {
			return false
		}
	}
	return true
}
