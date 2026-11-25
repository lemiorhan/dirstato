# dirstato — karar özeti (grilling, 2026-11-24)

## Ürün
- macOS'a özel, interaktif disk kullanım TUI'ı. Read-only, silme yok.
- Komut: `dirstato [path] [--min 100M]`. Path yoksa cwd.

## Tarama
- Go + Bubble Tea, paralel tarama.
- Boyut = disk kullanımı (allocated block). Hardlink bir kez sayılır.
- Söz: kökün toplamı `du -skx` ile tutar. Hardlink'in byte'larının hangi klasöre yazılacağı (deterministik kural) plana kalır.
- Kök bir symlink ise hedefi taranır. O zaman karşılaştırma `du -skx <path>/` iledir: sondaki `/` olmadan `du` symlink'in kendisini ölçer.
- Symlink takip edilmez, mount sınırı geçilmez.
- Tarama sırasında progress ekranı: dosya sayısı, toplam boyut, o anki path. Tree tarama bitince açılır.
- Okunamayan klasör `!` ile işaretlenir, status bar'da hata sayısı, devam edilir.
- Bütün node'lar hafızada. Hafıza bütçesi: home taramasında RSS ≤ 1 GB.

## Görünüm
- Açılıp kapanan tree + drill-down. Tab ile geçiş, seçim korunur.
- Açılışta kök açık, ilk seviye görünür.
- Kolonlar: boyut, parent'a göre %, bar, recursive dosya sayısı, alt tree'deki en yeni mtime.
- Dar terminalde önce tarih, sonra sayı düşer (genişlik eşiği knob).
- Dotfile'lar her zaman görünür.

## Sıralama ve filtre
- Varsayılan size desc, klasör/dosya karışık. `s`: size desc → size asc → isim.
- Min boyut görünüm filtresi: `--min` flag'i + `m` tuşu. 1024 tabanı, K/M/G.
- Gizlenenler için parent başına özet satırı.

## Etkileşim
- Keymap: ↑↓ j k, → l ⏎, ← h, Tab, s, m, c, r, ?, q.
- `c`: absolute path, gerekirse tek tırnakla quote'lanır, pbcopy.
- `r`: seçili klasörün alt tree'si yeniden taranır, parent toplamları güncellenir.

## Dağıtım
- Public repo `lemiorhan/dirstato`.
- Yeni tap `lemiorhan/homebrew-dirstato`, source'tan build eden formula.

## İlk tur kapsamı
- Uygulama: iki görünüm, boyut + dört kolon (%, bar, sayı, tarih), sıralama, filtre + özet satırı, kopyalama, rescan, progress ekranı.
- Testler: `du -skx` eval'i + path quote testi.
  - Review'dan sonra eklenenler: mount sınırı, fd sınırı ve rescan sonrası bellek testleri; `units` testleri; rescan ve NOTE düzeltmeleri için `internal/ui` testleri.
- Repo, tap ve release ayrı tur.
- İlk geliştirme adımı ölçüm:
  - Go'da ~/Code RSS'i → node başına byte. Home tahmini bundan çıkar; kesin rakam ilk gerçek `~` taramasında.
  - Go tarama süresi, baseline'lara karşı: paralel Node 29 s, `du` 49 s (sıcak cache).
  - Hafıza bütçeyi aşarsa R2-Q7 (görünüm filtresi, her şey hafızada) yeniden açılır.
  - Go `du`'dan yavaşsa paralel tarama tasarımı yeniden açılır.

## Kapanan ölçüm: Go tarama
- `DIRSTATO_EVAL_DIR=~/Code go test -run RealDir -v ./internal/scan` (2026-11-24):
  - tarama 11,6 s, aynı komutta `du` 46,5 s
  - kök boyutu `du` ile birebir: 38145683456 byte
  - 1,57M node, GC sonrası heap 183 MB (node başına 116 byte), peak RSS 288 MB
- Home'un node sayısı hâlâ bilinmiyor. RSS ≤ 1 GB bütçesi ilk gerçek `~` taramasında kesinleşir.

## Kapanan ölçüm
- 0,1 GB fark hardlink'ten geliyordu. (dev, inode) dedup ile `du` ile aynı çıktı:
  - python3 walk_dedup.py ~/Code → dup=9 disk=38.15GB
  - du -skx ~/Code → 37251512 KB ≈ 38.15 GB

## Varsayımlar (onaylandı, 2026-11-24)
1. Binary adı `dirstato`.
2. `c` sonrası status bar'da kısa "kopyalandı" mesajı.
3. Progress ekranında `q` / Ctrl-C taramayı iptal edip çıkar.
4. `r` bir dosyadayken parent klasörü tarar. Rescan sırasında UI küçük bir progress gösterip bekler.
5. `r` artık var olmayan bir öğedeyse node tree'den düşer, parent toplamları güncellenir.
6. Sıralama ve filtre bütün seviyelere aynı uygulanır.
7. Kök satırı eşiğin altında kalsa da gösterilir.
8. `m` input'u `500M`, `1.5G`, `0` (kapalı) alır. Geçersiz değer status bar'da hata verir.
9. Özet satırı her sıralamada listenin en altında durur.
10. Özet satırında `c`, `r` ve `→` parent klasöre uygulanır.
11. Özet satırı drill-down'da da görünür.
12. Tarih formatı YYYY-MM-DD.
13. Hata listesi için ayrı ekran yok.
14. Status bar: kök path, sıralama, eşik, görünüm modu, hata sayısı.
15. Ayarlar (sıralama, eşik, görünüm) çalıştırmalar arasında saklanmaz.
16. Hardlink'in deterministik atama kuralı plana bırakılır. Plan: tree içindeki link'lerden path'i sözlükte en küçük olan byte'ları taşır.
