# dirstato — ilk tur planı

Kararlar `docs/spec.md`'de. Bu dosya interface'leri, sırayı ve kabul kriterlerini
taşır.

## Bağımlılıklar

- Go 1.27 (`go.mod`), `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`,
  `charm.land/bubbles/v2` (textinput), `golang.org/x/sys/unix`.

## Paketler ve interface'ler

### `internal/scan`

- `Node`: `Name`, `Parent`, `Children`; kendi disk kullanımı ve kendi mtime'ı;
  alt tree toplamları (`Size`, `Files`, `Newest`); flag'ler (dir, unreadable).
  Full path saklanmaz, `Node.Path()` parent zincirinden kurar.
- `Tree`: `Root`, kökün device'ı, hardlink tablosu (`(dev, ino)` → node'lar),
  `Errors()`.
- `Scan(ctx, path, Options, *Progress) (*Tree, error)`: paralel tarama, bitince
  hardlink sahipliğini çözer ve toplamları hesaplar.
- `Progress`: atomic sayaçlar (dosya, klasör, byte) ve o an taranan path.
  Progress ekranı bunları okur.
- Rescan iki adım: `Tree.ScanSubtree(ctx, node, Options, *Progress)` arka planda
  yeni bir alt tree üretir ve tree'ye dokunmaz; `Tree.Apply(result)` UI
  goroutine'inde node'u yerinde günceller ya da (path artık yoksa) parent'tan
  çıkarır, hardlink tablosunu ve toplamları yeniden hesaplar. Node'un altındaki
  eski node'lar çöpe gider; UI tuttuğu node'ları (seçim, açık klasörler, cwd)
  path'leriyle yeni node'lara taşır.
- Kurallar:
  - Disk kullanımı = `st_blocks * 512`; dizinin kendi block'ları da sayılır.
  - lstat ile okunur, symlink takip edilmez.
  - Kökün device'ından farklı bir dizine inilmez.
  - Hardlink (nlink > 1, dir değil) byte'ları tek bir sahibe yazılır: tree
    içindeki link'ler arasında path'i sözlükte en küçük olan. Diğer link'ler
    listelenir, boyutu 0 olur.
  - Açılamayan dizin unreadable işaretlenir, kendi block'ları sayılır.
    Listelenip girdileri stat edilemeyen dizin de unreadable işaretlenir; o
    girdiler tree'ye girmez.
  - Bir worker bir dizinin fd'sini sadece o dizini okurken açık tutar; alt
    dizinlere inmeden kapatır. Açık fd sayısı worker sayısıyla sınırlı.
  - Dosyanın `Files` değeri 1, dizinin alt tree'deki dosya sayısı. Dizin
    sayılmaz.
  - `Newest`: alt tree'deki dosyaların en yeni mtime'ı; alt tree'de dosya yoksa
    dizinin kendi mtime'ı.

### `internal/units`

- `ParseSize(string) (int64, error)`: `500M`, `1.5G`, `100K`, `2T`, `0`, çıplak
  sayı = byte. 1024 tabanı, harf büyük/küçük fark etmez, sondaki `B`/`iB`
  kabul edilir.
- `FormatSize(int64) string`: `38.1G` gibi, 1024 tabanı.

### `internal/clip`

- `Quote(path) string`: güvenli karakterlerden oluşuyorsa olduğu gibi, değilse
  tek tırnak içinde (`'` → `'\''`).
- `Copy(text) error`: `pbcopy`'ye stdin ile yazar.

### `internal/ui`

- `New(Config) Model` — `Config`: kök path, başlangıç eşiği ve onun `--min`'e
  yazılan metni, copy fonksiyonu (test clipboard'a dokunmasın diye inject
  edilir). Eşik, kullanıcının yazdığı metinle birlikte saklanır; başlık ve `m`
  input'u o metni gösterir. Tarama `scan.Scan`'i
  doğrudan çağırır, UI testleri gerçek dosya sistemine dokunur.
- Durumlar: tarama (progress ekranı) → gezinme → (`r` ile) rescan → gezinme.
  `m` bir input açar, `?` yardım ekranını açar.
- Satırlar: node satırı ve özet satırı. Tree görünümü expand edilmiş node'ları
  derinlik öncelikli düzleştirir; drill-down tek dizinin çocuklarını gösterir.
  Sıralama ve filtre satırlar kurulurken uygulanır.
- Ekran: üstte başlık (kök, sıralama, eşik, görünüm, hata sayısı), ortada
  satırlar, altta R3-Q1 preview'undaki tuş açıklaması ve status mesajı.
- Kolonlar: boyut, %, bar, dosya sayısı, tarih. Genişlik knob'larının altında
  önce tarih, sonra sayı düşer.

### `main.go`

- `dirstato [path] [--min SIZE]`, flag'ler path'ten önce ya da sonra gelebilir.
  Path yoksa cwd. Path dizin değilse hata verip çıkar.

## Sıra

1. `units`, `clip` + testleri.
2. `scan` + fixture eval'i + rescan eval'i.
3. Ölçüm kapısı (gate): `~/Code` üstünde. Geçmezse dur, kullanıcıya git.
4. `ui` + `main.go`, tmux ile elle doğrulama.
5. `review-gate`.

## Kabul kriterleri

- **AC1 (eval, fixture):** iç içe dizinler, aynı dizinde ve farklı dizinlerde
  hardlink çiftleri, symlink, sparse dosya, açılamayan bir dizin ve
  listelenip girdileri stat edilemeyen bir dizin. Kökün `Size`'ı
  `BLOCKSIZE=512 /usr/bin/du -sx` çıktısının 512 katıyla birebir tutar. 512
  byte'lık block'la karşılaştırıldığı için yuvarlama yok. İki dizin de
  işaretlidir, `Errors()` 2'dir.
- **AC2 (eval, rescan):** dosya silinir ve alt tree yeniden taranır, toplam
  taze bir `du` ile tutar. Hardlink'in byte'larını taşıyan link silinince kalan
  link byte'ları alır. Silinmiş bir dizin rescan edilince node tree'den düşer.
- **AC3 (opt-in eval):** `DIRSTATO_EVAL_DIR` verilirse o klasör taranır; dosya,
  dizin, hata sayısı, süre, GC sonrası heap, peak RSS ve node başına byte
  raporlanır; `BLOCKSIZE=512 /usr/bin/du -sx` ile fark raporlanır (canlı tree,
  birebir değil). Peak RSS'in birimi ilk koşuda `/usr/bin/time -l` ile
  karşılaştırılarak doğrulanır.
- **AC4 (ölçüm kapısı):** `~/Code` üstünde AC3 ve `/usr/bin/du -skx` aynı
  komutta, art arda çalışır.
  - Hafıza: node başına byte'tan "1 GB'a sığan node sayısı" çıkar. Home'un
    node sayısı bilinmiyor; kesin cevap ilk gerçek `~` taramasında.
  - Hız: Go taraması aynı komuttaki `du`'dan yavaş olmamalı.
  - Biri geçmezse dur, R2-Q7 ya da tarama tasarımı kullanıcıyla yeniden açılır.
- **AC5 (quote):** elle yazılmış beklenen değerler + her örneğin `/bin/sh`
  üstünden round-trip'i.
- **AC6:** `go build ./...`, `go vet ./...`, `go test ./...` geçer.
- **AC7 (elle):** tmux içinde gerçek bir klasörde: tree, drill-down, Tab
  geçişi, `s` döngüsü, `m` ile eşik, özet satırı, `c` ile kopyalama, `r` ile
  rescan. Kullanıcının clipboard'ı doğrulamadan önce saklanır, sonra geri
  konur.
