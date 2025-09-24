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
- `-ns-only` - применить только подавление шума (без эхоподавления, speaker файл не нужен)
- `-aec-only` - применить только эхоподавление (без подавления шума)
- `-help` - показать справку

### Настройки шумодава

Дополнительные параметры для тонкой настройки подавления шума:

- `-noise-suppress` - уровень подавления шума в дБ (по умолчанию: -15.0, более отрицательные значения = больше подавления)
- `-vad` - включить обнаружение голосовой активности (Voice Activity Detection)
- `-vad-prob-start` - порог вероятности начала речи для VAD, 0-100 (по умолчанию: 80)
- `-vad-prob-continue` - порог вероятности продолжения речи для VAD, 0-100 (по умолчанию: 65)
- `-agc` - включить автоматическую регулировку усиления (Automatic Gain Control)
- `-agc-level` - целевой RMS уровень для AGC (по умолчанию: 30000.0)

### Компенсация задержки

Опция `-prev-speaker` использует предыдущий фрейм speaker с текущим фреймом microphone. Это полезно для компенсации задержки обработки в системах реального времени:

```bash
# Обычный режим: mic[n] + speaker[n] -> output[n]
./speex -mic mic.alaw -speaker spk.alaw -output normal.alaw

# Режим с компенсацией: mic[n] + speaker[n-1] -> output[n]  
./speex -mic mic.alaw -speaker spk.alaw -output delayed.alaw -prev-speaker
```

### Режимы обработки

Доступно 4 режима обработки аудио:

```bash
# 1. AEC → NS (по умолчанию): эхоподавление, затем шумоподавление
./speex -mic mic.alaw -speaker spk.alaw -output aec_first.alaw

# 2. NS → AEC: шумоподавление, затем эхоподавление  
./speex -mic mic.alaw -speaker spk.alaw -output ns_first.alaw -ns-first

# 3. Только NS: только шумоподавление (speaker файл не нужен)
./speex -mic mic.alaw -output ns_only.alaw -ns-only

# 4. Только AEC: только эхоподавление (speaker файл обязателен)
./speex -mic mic.alaw -speaker spk.alaw -output aec_only.alaw -aec-only

# Дополнительно: любой режим с компенсацией задержки (кроме NS-only)
./speex -mic mic.alaw -speaker spk.alaw -output delayed.alaw -aec-only -prev-speaker
```

**Когда использовать NS-first:**
- Высокий уровень фонового шума в микрофоне
- Необходимость улучшить качество сигнала перед эхоподавлением
- Экспериментальные настройки для конкретных сценариев

### Режим только подавления шума

Опция `-ns-only` применяет только подавление шума без эхоподавления. В этом режиме speaker файл не требуется:

```bash
# Только подавление шума (speaker файл не нужен)
./speex -mic noisy_audio.alaw -output clean_audio.alaw -ns-only
```

**Когда использовать NS-only:**
- Обработка записей без эха (студийные записи, интервью)
- Удаление фонового шума из аудиофайлов
- Предобработка аудио перед другими алгоритмами
- Быстрая очистка от шума без AEC overhead'а

### Режим только эхоподавления

Опция `-aec-only` применяет только эхоподавление без подавления шума. Speaker файл обязателен:

```bash
# Только эхоподавление (speaker файл обязателен)
./speex -mic mic_with_echo.alaw -speaker reference.alaw -output clean.alaw -aec-only

# AEC с компенсацией задержки
./speex -mic mic.alaw -speaker ref.alaw -output clean.alaw -aec-only -prev-speaker
```

**Когда использовать AEC-only:**
- Чистые записи с эхом (без фонового шума)
- Телефонные/VoIP звонки с эхом
- Когда нужно сохранить оригинальные характеристики звука
- Предобработка для других алгоритмов шумоподавления

### Примеры настройки шумодава

```bash
# Стандартное подавление шума
./speex -mic noisy.alaw -output clean.alaw -ns-only

# Агрессивное подавление шума (может ухудшить качество речи)
./speex -mic noisy.alaw -output clean.alaw -ns-only -noise-suppress -25

# Мягкое подавление шума для сохранения качества речи
./speex -mic noisy.alaw -output clean.alaw -ns-only -noise-suppress -8

# С включенным VAD (детектор речи)
./speex -mic noisy.alaw -output clean.alaw -ns-only -vad -vad-prob-start 85

# С автоматической регулировкой громкости
./speex -mic quiet.alaw -output loud.alaw -ns-only -agc -agc-level 35000

# Полная конфигурация: NS + VAD + AGC
./speex -mic input.alaw -output output.alaw -ns-only \
  -noise-suppress -12 -vad -vad-prob-start 75 -vad-prob-continue 60 \
  -agc -agc-level 28000
```

### Рекомендации по настройке

**Уровень подавления шума (`-noise-suppress`):**
- `-5` до `-10` - легкое подавление, хорошее качество речи
- `-15` - стандартное (по умолчанию), баланс качества и очистки
- `-20` до `-30` - агрессивное, может исказить речь

**VAD настройки:**
- `vad-prob-start` 70-85 - чувствительность детектора начала речи
- `vad-prob-continue` 60-70 - чувствительность продолжения речи
- Высокие значения = менее чувствительный детектор

**AGC настройки:**
- 20000-40000 - типичный диапазон RMS уровней
- Меньше значение = тише выход, больше = громче выход

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
