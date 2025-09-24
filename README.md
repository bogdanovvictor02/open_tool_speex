# Speex AEC Console Tool

Консольное приложение для эхоподавления (AEC) с использованием библиотеки SpeexDSP.

## Возможности

- Эхоподавление (Echo Cancellation) 
- Подавление шума (Noise Suppression)
- Поддержка формата A-law PCM 16 кГц моно
- Обработка кадров по 320 сэмплов (20 мс)
- Echo tail 200 мс (3200 сэмплов)

## Требования

- macOS с Homebrew
- Go 1.24+
- SpeexDSP library

## Установка зависимостей

```bash
brew install speexdsp pkg-config
```

## Сборка

```bash
go build
```

## Использование

```bash
./speex -mic microphone.alaw -speaker speaker.alaw -output clean_output.alaw
```

### Параметры

- `-mic` - входной файл с микрофона (raw A-law, 16 кГц моно)
- `-speaker` - референсный файл с динамика (raw A-law, 16 кГц моно) 
- `-output` - выходной файл (по умолчанию: output.alaw)
- `-prev-speaker` - использовать предыдущий фрейм speaker для компенсации задержки
- `-ns-first` - применить подавление шума перед эхоподавлением (по умолчанию: AEC → NS)
- `-help` - показать справку

### Компенсация задержки

Опция `-prev-speaker` использует предыдущий фрейм speaker с текущим фреймом microphone. Это полезно для компенсации задержки обработки в системах реального времени:

```bash
# Обычный режим: mic[n] + speaker[n] -> output[n]
./speex -mic mic.alaw -speaker spk.alaw -output normal.alaw

# Режим с компенсацией: mic[n] + speaker[n-1] -> output[n]  
./speex -mic mic.alaw -speaker spk.alaw -output delayed.alaw -prev-speaker
```

### Порядок обработки

По умолчанию применяется **AEC → NS** (эхоподавление, затем подавление шума). Опция `-ns-first` меняет порядок на **NS → AEC**:

```bash
# Обычный порядок: AEC → NS (по умолчанию)
./speex -mic mic.alaw -speaker spk.alaw -output aec_first.alaw

# Обратный порядок: NS → AEC
./speex -mic mic.alaw -speaker spk.alaw -output ns_first.alaw -ns-first

# Комбинация: NS → AEC + компенсация задержки  
./speex -mic mic.alaw -speaker spk.alaw -output combined.alaw -ns-first -prev-speaker
```

**Когда использовать NS-first:**
- Высокий уровень фонового шума в микрофоне
- Необходимость улучшить качество сигнала перед эхоподавлением
- Экспериментальные настройки для конкретных сценариев

## Формат файлов

Входные и выходные файлы должны быть в формате:
- Raw A-law PCM (без заголовков)
- 16 кГц частота дискретизации
- Моно (1 канал)
- 8 бит на сэмпл

## Создание тестовых файлов

### Из WAV в A-law:
```bash
# С помощью FFmpeg
ffmpeg -i input.wav -ar 16000 -ac 1 -f alaw output.alaw

# С помощью SoX  
sox input.wav -t al -r 16000 -c 1 output.alaw
```

### Из A-law в WAV:
```bash
# С помощью FFmpeg
ffmpeg -f alaw -ar 16000 -ac 1 -i input.alaw output.wav

# С помощью SoX
sox -t al -r 16000 -c 1 input.alaw output.wav
```

## Архитектура

- `main.go` - CLI интерфейс и основной цикл обработки
- `alaw.go` - A-law кодек (кодирование/декодирование)
- `speex_aec.go` - cgo обертка для SpeexDSP AEC/NS

## Производительность

- Обработка в реальном времени на современных системах
- Прогресс выводится каждые ~16 секунд
- Потребление памяти: ~10 МБ

## Примеры использования

### Базовое эхоподавление
```bash
./speex -mic recorded_call.alaw -speaker playback_reference.alaw
```

### С указанием выходного файла
```bash
./speex -mic mic.alaw -speaker spk.alaw -output cleaned.alaw
```

## Лимитации

- Файлы должны быть синхронизированы по времени
- Поддерживается только моно A-law 16 кГц
- Максимальная длина echo tail: 200 мс
