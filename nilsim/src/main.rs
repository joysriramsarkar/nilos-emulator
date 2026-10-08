// nilsim/src/main.rs — Onuron OS Desktop Mobile Simulator (Alap Framework & Figma 330x680)
#![allow(dead_code, unused_variables, unused_imports)]

mod audio;
use audio::AudioEngine;

mod nilvm;
use nilvm::NilVM;

#[cfg(target_os = "windows")]
mod nilbrowser;

#[cfg(target_os = "windows")]
#[link(name = "user32")]
extern "system" {
    fn SetFocus(hWnd: isize) -> isize;
}
// - VLC Media Player (org.videolan.vlc): Full Official VideoLAN Media Engine for NilOS
// - NilZar Browser: Full Declarative Web Browser with History & Bookmarks Drawers, Real Web Fetcher & Portals
// - Indian Standard Time (IST UTC+5:30)
// - Full Bengali Dari '।' (\u{0964}) and Double Dari '॥' (\u{0965})
// - Package Manager (nilpkg) with real CLI & search/install/remove
// - SoftBus Distributed Device Mesh with File Drop & Clipboard Sync
// - Smartphone Launcher with 4-Column Squircle App Grid & Translucent Bottom Dock
// - ArkTS Smart Notes, Calculator & Music Player
// - Dynamic Island & Control Center
// - Segment-based HarfBuzz OpenType typography & Linux GNU Bash Terminal

use std::fs::{self};
use std::path::{Path, PathBuf};
use std::process::Command;
use std::sync::{Arc, Mutex};
use std::time::{SystemTime, UNIX_EPOCH};

use fontdue::{Font, FontSettings};
use minifb::{Key, KeyRepeat, MouseButton, MouseMode, Scale, Window, WindowOptions};

// ─── Smartphone Dimensions ───────────────────────────────────────────────────
const SCREEN_WIDTH: usize = 330;
const SCREEN_HEIGHT: usize = 680;
const HOME_SCREEN_RAW: &[u8] = include_bytes!("../assets/home_screen.bin");
const WALLPAPER_RAW: &[u8] = include_bytes!("../assets/wallpaper.bin");

// ─── Material Design 3 (Material You / M3 Dark Palette) ───────────────────────────
pub const M3_SURFACE: u32 = 0x141218;                  // MD3 Baseline Dark Surface
pub const M3_SURFACE_DIM: u32 = 0x141218;
pub const M3_SURFACE_BRIGHT: u32 = 0x3B383E;
pub const M3_SURFACE_CONTAINER_LOWEST: u32 = 0x0F0D13;
pub const M3_SURFACE_CONTAINER_LOW: u32 = 0x1D1B20;    // Low elevation cards
pub const M3_SURFACE_CONTAINER: u32 = 0x211F26;        // Standard M3 cards & app bars
pub const M3_SURFACE_CONTAINER_HIGH: u32 = 0x2B2930;   // Elevated cards, dialogs, drawers
pub const M3_SURFACE_CONTAINER_HIGHEST: u32 = 0x36343B;// Chips, hover/active tiles
pub const M3_ON_SURFACE: u32 = 0xE6E0E9;               // High-emphasis text (90% white)
pub const M3_ON_SURFACE_VARIANT: u32 = 0xCAC4D0;       // Medium-emphasis subtitles/captions
pub const M3_OUTLINE: u32 = 0x938F99;                  // Component borders
pub const M3_OUTLINE_VARIANT: u32 = 0x49454F;          // Subtle dividers / card outlines

// M3 Dynamic Tonal Accents (Lavender / Purple Violet Material You Palette)
pub const M3_PRIMARY: u32 = 0xD0BCFF;                  // M3 Primary Accent (Lilac/Lavender)
pub const M3_ON_PRIMARY: u32 = 0x381E72;               // Text on Primary
pub const M3_PRIMARY_CONTAINER: u32 = 0x4F378B;        // Filled button / active nav pill
pub const M3_ON_PRIMARY_CONTAINER: u32 = 0xEADDFF;     // Text on Primary Container

pub const M3_SECONDARY: u32 = 0xCCC2DC;                // M3 Secondary (Soft Slate/Mauve)
pub const M3_ON_SECONDARY: u32 = 0x332D41;
pub const M3_SECONDARY_CONTAINER: u32 = 0x4A4458;      // Filter chips & segmented buttons
pub const M3_ON_SECONDARY_CONTAINER: u32 = 0xE8DEF8;

pub const M3_TERTIARY: u32 = 0xEFB8C8;                 // M3 Tertiary (Warm Rose / Peach)
pub const M3_ON_TERTIARY: u32 = 0x492532;
pub const M3_TERTIARY_CONTAINER: u32 = 0x633B48;
pub const M3_ON_TERTIARY_CONTAINER: u32 = 0xFFD8E4;

pub const M3_ERROR: u32 = 0xF2B8B8;
pub const M3_ON_ERROR: u32 = 0x601410;
pub const M3_ERROR_CONTAINER: u32 = 0x8C1D18;
pub const M3_ON_ERROR_CONTAINER: u32 = 0xF9DEDC;

// Expressive tonal accents for apps
pub const M3_TEAL: u32 = 0x8CD6CF;
pub const M3_BLUE: u32 = 0xA8C7FA;
pub const M3_GREEN: u32 = 0x85E0A3;
pub const M3_AMBER: u32 = 0xFDE293;
pub const M3_ORANGE: u32 = 0xFFB787;
pub const M3_PINK: u32 = 0xFFB0CD;

// Legacy Compatibility Aliases (re-routed to M3 tokens)
const COLOR_BG: u32 = M3_SURFACE;
const COLOR_DOCK_BG: u32 = M3_SURFACE_CONTAINER;
const COLOR_SURFACE: u32 = M3_SURFACE_CONTAINER;
const COLOR_SURFACE_ALT: u32 = M3_SURFACE_CONTAINER_HIGH;
const COLOR_BORDER: u32 = M3_OUTLINE_VARIANT;
const COLOR_CYAN: u32 = M3_TEAL;
const COLOR_BLUE: u32 = M3_BLUE;
const COLOR_GREEN: u32 = M3_GREEN;
const COLOR_AMBER: u32 = M3_AMBER;
const COLOR_PURPLE: u32 = M3_PRIMARY;
const COLOR_RED: u32 = M3_ERROR;
const COLOR_FOX: u32 = M3_ORANGE;
const COLOR_VLC: u32 = 0xFF9E66;
const COLOR_TEXT_HIGH: u32 = M3_ON_SURFACE;
const COLOR_TEXT_MED: u32 = M3_ON_SURFACE_VARIANT;
const COLOR_TEXT_DIM: u32 = 0x938F99;
const COLOR_ACCENT_BG: u32 = M3_PRIMARY_CONTAINER;
const COLOR_TERM_BG: u32 = 0x100E14;
const COLOR_NANO_HDR: u32 = M3_PRIMARY_CONTAINER;
const COLOR_GOLD: u32 = M3_AMBER;
const COLOR_TEAL: u32 = M3_TEAL;
const COLOR_PINK: u32 = M3_PINK;

// ─── Screen Enum ──────────────────────────────────────────────────────────────
#[derive(Debug, Clone, PartialEq)]
enum Screen {
    Lockscreen,
    Home,
    AppPhone,
    AppMessages,
    AppFiles,
    AppSettings,
    AppNilPkg,
    AppSoftBus,
    AppAndroid,
    AppTerminal,
    AppNotes,
    AppCalculator,
    AppMusic,
    AppBrowser,
    ControlCenter,
    NanoEditor,
    TaskSwitcher,
}

// ─── Interactive Touch/Click Button ──────────────────────────────────────────
#[derive(Clone, Debug)]
struct TouchButton {
    x: i16,
    y: i16,
    w: u16,
    h: u16,
    id: String,
}

impl TouchButton {
    fn contains(&self, px: i16, py: i16) -> bool {
        px >= self.x && px < self.x + self.w as i16 && py >= self.y && py < self.y + self.h as i16
    }
}

// ─── Indian Standard Time (IST UTC+5:30) & Bengali Digits ─────────────────────
fn get_ist_time_str() -> String {
    let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap_or_default().as_secs();
    let ist_secs = now + 19800; // UTC+5:30 (India Standard Time)
    let hours = ((ist_secs / 3600) % 24) as u8;
    let mins = ((ist_secs / 60) % 60) as u8;
    let raw = format!("{:02}:{:02}", hours, mins);
    to_bengali_digits(&raw)
}

fn to_bengali_digits(s: &str) -> String {
    s.chars()
        .map(|c| match c {
            '0' => '০',
            '1' => '১',
            '2' => '২',
            '3' => '৩',
            '4' => '৪',
            '5' => '৫',
            '6' => '৬',
            '7' => '৭',
            '8' => '৮',
            '9' => '৯',
            _ => c,
        })
        .collect()
}

// ─── Bengali Unicode Character Checker ────────────────────────────────────────
#[inline(always)]
fn is_bengali_char(ch: char) -> bool {
    (ch >= '\u{0980}' && ch <= '\u{09FF}')
        || (ch >= '\u{0964}' && ch <= '\u{0965}') // Bengali Dari '।' and '॥'
        || (ch >= '\u{200C}' && ch <= '\u{200D}') // ZWNJ and ZWJ
}

#[inline(always)]
fn is_valid_input_char(ch: char) -> bool {
    (ch >= ' ' && ch <= '~') || is_bengali_char(ch) || ch == '\n' || ch == '\t'
}

// ─── Typography Engine with Segment-Based Script Itemization ──────────────────
struct FontEngine {
    ui_font: Font,
    mono_font: Font,
    bengali_font: Font,
    bengali_face_bytes: Vec<u8>,
}

impl FontEngine {
    fn load() -> Self {
        let ui_bytes = fs::read(r"C:\Windows\Fonts\segoeui.ttf")
            .or_else(|_| fs::read(r"C:\Windows\Fonts\arial.ttf"))
            .or_else(|_| fs::read(r"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"))
            .expect("Could not find system UI font");

        let mono_bytes = fs::read(r"C:\Windows\Fonts\consola.ttf")
            .or_else(|_| fs::read(r"C:\Windows\Fonts\cour.ttf"))
            .unwrap_or_else(|_| ui_bytes.clone());

        let bengali_bytes = fs::read(r"C:\Windows\Fonts\vrinda.ttf")
            .or_else(|_| fs::read(r"C:\Windows\Fonts\Shonar.ttf"))
            .or_else(|_| fs::read(r"C:\Windows\Fonts\vrindab.ttf"))
            .or_else(|_| fs::read(r"C:\Windows\Fonts\Shonarb.ttf"))
            .unwrap_or_else(|_| ui_bytes.clone());

        let ui_font = Font::from_bytes(ui_bytes, FontSettings::default())
            .expect("Failed to parse UI font");
        let mono_font = Font::from_bytes(mono_bytes, FontSettings::default())
            .expect("Failed to parse Monospace font");
        let bengali_font = Font::from_bytes(bengali_bytes.clone(), FontSettings::default())
            .expect("Failed to parse Bengali font");

        FontEngine {
            ui_font,
            mono_font,
            bengali_font,
            bengali_face_bytes: bengali_bytes,
        }
    }
}

// ─── Alpha Blending Helper ───────────────────────────────────────────────────
#[inline(always)]
fn blend_pixel(bg: u32, fg: u32, alpha: u8) -> u32 {
    if alpha == 0 {
        return bg;
    }
    if alpha == 255 {
        return fg;
    }
    let a = alpha as u32;
    let inv_a = 255 - a;

    let r_bg = (bg >> 16) & 0xFF;
    let g_bg = (bg >> 8) & 0xFF;
    let b_bg = bg & 0xFF;

    let r_fg = (fg >> 16) & 0xFF;
    let g_fg = (fg >> 8) & 0xFF;
    let b_fg = fg & 0xFF;

    let r = (r_fg * a + r_bg * inv_a) / 255;
    let g = (g_fg * a + g_bg * inv_a) / 255;
    let b = (b_fg * a + b_bg * inv_a) / 255;

    (r << 16) | (g << 8) | b
}

// ─── NilOS Virtual Filesystem & Path Normalizer ─────────────────────────────────
fn ensure_nilos_storage() -> (PathBuf, PathBuf) {
    let current_dir = std::env::current_dir().unwrap_or_else(|_| PathBuf::from("."));
    let exe_dir = std::env::current_exe()
        .ok()
        .and_then(|p| p.parent().map(|d| d.to_path_buf()))
        .unwrap_or_else(|| PathBuf::from("."));

    let candidate_storage = if current_dir.join("storage").is_dir() {
        current_dir.join("storage")
    } else if exe_dir.join("storage").is_dir() {
        exe_dir.join("storage")
    } else if exe_dir
        .parent()
        .and_then(|p| p.parent())
        .map(|p| p.join("storage"))
        .map(|p| p.is_dir())
        .unwrap_or(false)
    {
        exe_dir.parent().unwrap().parent().unwrap().join("storage")
    } else {
        current_dir.join("storage")
    };

    let storage_root = if let Ok(canon) = candidate_storage.canonicalize() {
        canon
    } else {
        let _ = fs::create_dir_all(&candidate_storage);
        candidate_storage.canonicalize().unwrap_or(candidate_storage)
    };

    let home_dir = storage_root.join("home").join("joy");
    let docs_dir = home_dir.join("Documents");
    let music_dir = home_dir.join("Music");
    let videos_dir = home_dir.join("Videos");
    let downloads_dir = home_dir.join("Downloads");
    let pictures_dir = home_dir.join("Pictures");
    let etc_dir = storage_root.join("etc");
    let bin_dir = storage_root.join("bin");
    let var_log_dir = storage_root.join("var").join("log");
    let tmp_dir = storage_root.join("tmp");

    let _ = fs::create_dir_all(&docs_dir);
    let _ = fs::create_dir_all(&music_dir);
    let _ = fs::create_dir_all(&videos_dir);
    let _ = fs::create_dir_all(&downloads_dir);
    let _ = fs::create_dir_all(&pictures_dir);
    let _ = fs::create_dir_all(&etc_dir);
    let _ = fs::create_dir_all(&bin_dir);
    let _ = fs::create_dir_all(&var_log_dir);
    let _ = fs::create_dir_all(&tmp_dir);

    // Populate essential NilOS files if missing
    let os_rel = etc_dir.join("os-release");
    if !os_rel.exists() {
        let _ = fs::write(
            &os_rel,
            "NAME=\"NilOS\"\nVERSION=\"1.7.0 (VLC & NilZar Edition)\"\nID=nilos\nID_LIKE=debian\nPRETTY_NAME=\"NilOS 1.7.0 (GNU/Linux nilkernel 6.6.21-nil-aarch64)\"\nHOME_URL=\"https://nilos.dev\"\n",
        );
    }

    let hosts_file = etc_dir.join("hosts");
    if !hosts_file.exists() {
        let _ = fs::write(
            &hosts_file,
            "127.0.0.1   localhost onuron\n::1         localhost ip6-localhost ip6-loopback\n192.168.1.100 nildevice.local\n",
        );
    }

    let passwd_file = etc_dir.join("passwd");
    if !passwd_file.exists() {
        let _ = fs::write(
            &passwd_file,
            "root:x:0:0:root:/root:/bin/bash\njoy:x:1000:1000:জয় সরকার:/home/joy:/bin/bash\nnil:x:999:999:NilOS System Daemon:/var/nil:/sbin/nologin\n",
        );
    }

    let notes_file = docs_dir.join("notes.md");
    if !notes_file.exists() {
        let _ = fs::write(
            &notes_file,
            "# অনুরণ ওএস (Onuron OS) ব্যবহার নির্দেশিকা\n\n- টার্মিনাল: `ls -la`, `pwd`, `cat`, `nano`, `uname -a`, `neofetch`, `free -h`\n- ফাইলস অ্যাপ: ফোল্ডারে ক্লিক করে ভিতরে যান\n- ভিএলসি প্লেয়ার: আসল গান ও ভিডিও চালানো যায়\n- নীলপ্যাক: নতুন অ্যাপস ইনস্টল করুন\n",
        );
    }

    let changelog_file = docs_dir.join("CHANGELOG.txt");
    if !changelog_file.exists() {
        let _ = fs::write(
            &changelog_file,
            "Onuron OS Version 1.7.0 Changelog\n==============================\n[+] Virtual Isolated Linux Filesystem (/home/joy)\n[+] VLC Media Player Integration with live audio/video\n[+] NilZar Web Browser with live HTTP parsing\n[+] Bengali Language & Bangla typography support\n[+] High-performance 5-column responsive app launcher\n",
        );
    }

    let update_script = docs_dir.join("update.sh");
    if !update_script.exists() {
        let _ = fs::write(
            &update_script,
            "#!/bin/bash\necho \"[NilPkg] প্যাকেজ রিপোজিটরি সিঙ্ক হচ্ছে...\"\necho \"[NilPkg] সিস্টেম কার্নেল nilkernel 6.6.21 আপ-টু-ডেট।\"\necho \"[NilPkg] সমস্ত অ্যাপ সফলভাবে আপডেট সম্পন্ন!\"\n",
        );
    }

    let gonona_file = home_dir.join("gonona.nil");
    if !gonona_file.exists() {
        let _ = fs::write(
            &gonona_file,
            "// gonona.nil — Onuron OS গণক ও ভেক্টর অপ্টিমাইজার\nvar name = \"অনুরণ গণক\"\nvar base = 15\nvar multiplier = 4\nvar total = base * multiplier\nprint(\"প্রোগ্রাম নাম: \", name)\nprint(\"মোট ফলাফল: \", total)\nfor i in 1..5 {\n    var square = i * i\n    print(\"সংখ্যা: \", i, \" -> বর্গ: \", square)\n}\nsend(\"system_monitor\", \"গণনা সমাপ্ত\")\n",
        );
    }

    let alap_file = home_dir.join("alap_counter.nil");
    if !alap_file.exists() {
        let _ = fs::write(
            &alap_file,
            "// alap_counter.nil — Alap Reactive UI কাউন্টার স্ক্রিপ্ট\nvar counter = 0\nfor step in 1..8 {\n    counter = counter + 10\n    print(\"কাউন্টার ধাপ: \", step, \" -> বর্তমান মান: \", counter)\n}\nsend(\"ui_bus\", \"কাউন্টার আপডেট সম্পন্ন\")\n",
        );
    }

    let mesh_file = home_dir.join("softbus_mesh.nil");
    if !mesh_file.exists() {
        let _ = fs::write(
            &mesh_file,
            "// softbus_mesh.nil — SoftBus ডিস্ট্রিবিউটেড নোড পিং টেস্ট\nvar node1 = \"Desktop-Host\"\nvar node2 = \"NilPhone-Ultra\"\nvar latency_ms = 4\nprint(\"নোড সংযোগ স্থাপিত: \", node1, \" <-> \", node2)\nprint(\"নেটওয়ার্ক ল্যাটেন্সি: \", latency_ms, \" ms\")\nsend(\"softbus_router\", \"Heartbeat Ping ACK\")\n",
        );
    }

    (storage_root, home_dir)
}

fn virtual_to_disk_path(storage_root: &Path, current_disk_dir: &Path, input: &str) -> PathBuf {
    let input = input.trim();
    let home_dir = storage_root.join("home").join("joy");
    let target = if input.is_empty() || input == "~" {
        home_dir
    } else if input.starts_with("~/") {
        home_dir.join(&input[2..])
    } else if input == "/" {
        storage_root.to_path_buf()
    } else if input.starts_with('/') {
        storage_root.join(input.trim_start_matches('/'))
    } else {
        current_disk_dir.join(input)
    };

    let mut normalized = PathBuf::new();
    for comp in target.components() {
        match comp {
            std::path::Component::Prefix(p) => normalized.push(p.as_os_str()),
            std::path::Component::RootDir => normalized.push(std::path::MAIN_SEPARATOR.to_string()),
            std::path::Component::CurDir => {}
            std::path::Component::ParentDir => {
                if normalized != storage_root && normalized.starts_with(storage_root) {
                    normalized.pop();
                }
            }
            std::path::Component::Normal(c) => normalized.push(c),
        }
    }

    if !normalized.starts_with(storage_root) {
        storage_root.to_path_buf()
    } else {
        normalized
    }
}

fn disk_to_virtual_display(storage_root: &Path, disk_path: &Path) -> String {
    let home_dir = storage_root.join("home").join("joy");
    if disk_path == home_dir {
        "/home/joy".to_string()
    } else if let Ok(rel) = disk_path.strip_prefix(&home_dir) {
        let s = rel.to_string_lossy().replace('\\', "/");
        if s.is_empty() {
            "/home/joy".to_string()
        } else {
            format!("/home/joy/{}", s)
        }
    } else if disk_path == storage_root {
        "/".to_string()
    } else if let Ok(rel) = disk_path.strip_prefix(storage_root) {
        let s = rel.to_string_lossy().replace('\\', "/");
        format!("/{}", s)
    } else {
        "/home/joy".to_string()
    }
}

fn disk_to_prompt_display(storage_root: &Path, disk_path: &Path) -> String {
    let home_dir = storage_root.join("home").join("joy");
    if disk_path == home_dir {
        "~".to_string()
    } else if let Ok(rel) = disk_path.strip_prefix(&home_dir) {
        let s = rel.to_string_lossy().replace('\\', "/");
        if s.is_empty() {
            "~".to_string()
        } else {
            format!("~/{}", s)
        }
    } else if disk_path == storage_root {
        "/".to_string()
    } else if let Ok(rel) = disk_path.strip_prefix(storage_root) {
        let s = rel.to_string_lossy().replace('\\', "/");
        format!("/{}", s)
    } else {
        "~".to_string()
    }
}

fn clean_normalize_path(base: &Path, rel: &str) -> PathBuf {
    virtual_to_disk_path(base, base, rel)
}


// ─── Framebuffer Painter ──────────────────────────────────────────────────────
struct FramePainter<'a> {
    buffer: &'a mut [u32],
    width: usize,
    height: usize,
    fonts: &'a FontEngine,
    buttons: Vec<TouchButton>,
}

impl<'a> FramePainter<'a> {
    fn new(buffer: &'a mut [u32], width: usize, height: usize, fonts: &'a FontEngine) -> Self {
        FramePainter {
            buffer,
            width,
            height,
            fonts,
            buttons: Vec::new(),
        }
    }

    fn blend_pixel_at(&mut self, px: i16, py: i16, color: u32, alpha: u8) {
        if alpha == 0 || px < 0 || px >= self.width as i16 || py < 0 || py >= self.height as i16 {
            return;
        }
        let idx = (py as usize) * self.width + (px as usize);
        self.buffer[idx] = blend_pixel(self.buffer[idx], color, alpha);
    }

    fn fill_rect(&mut self, x: i16, y: i16, w: u16, h: u16, color: u32) {
        for dy in 0..(h as i16) {
            let py = y + dy;
            if py < 0 || py >= self.height as i16 {
                continue;
            }
            let row_offset = (py as usize) * self.width;
            for dx in 0..(w as i16) {
                let px = x + dx;
                if px >= 0 && px < self.width as i16 {
                    self.buffer[row_offset + px as usize] = color;
                }
            }
        }
    }

    fn fill_rounded_rect(&mut self, x: i16, y: i16, w: u16, h: u16, r: i16, color: u32) {
        let r = r.min((w / 2) as i16).min((h / 2) as i16).max(0);
        if r <= 0 {
            self.fill_rect(x, y, w, h, color);
            return;
        }
        let r_f = r as f32;

        for dy in 0..(h as i16) {
            let py = y + dy;
            if py < 0 || py >= self.height as i16 {
                continue;
            }
            let row_offset = (py as usize) * self.width;
            for dx in 0..(w as i16) {
                let px = x + dx;
                if px < 0 || px >= self.width as i16 {
                    continue;
                }

                let (cx, cy) = if dx < r && dy < r {
                    (r_f - (dx as f32 + 0.5), r_f - (dy as f32 + 0.5))
                } else if dx >= (w as i16 - r) && dy < r {
                    ((dx as f32 + 0.5) - (w as f32 - r_f), r_f - (dy as f32 + 0.5))
                } else if dx < r && dy >= (h as i16 - r) {
                    (r_f - (dx as f32 + 0.5), (dy as f32 + 0.5) - (h as f32 - r_f))
                } else if dx >= (w as i16 - r) && dy >= (h as i16 - r) {
                    ((dx as f32 + 0.5) - (w as f32 - r_f), (dy as f32 + 0.5) - (h as f32 - r_f))
                } else {
                    (-1.0, -1.0)
                };

                let idx = row_offset + px as usize;
                if cx >= 0.0 {
                    let dist = (cx * cx + cy * cy).sqrt();
                    if dist <= r_f - 0.5 {
                        self.buffer[idx] = color;
                    } else if dist < r_f + 0.5 {
                        let alpha = ((r_f + 0.5 - dist) * 255.0).clamp(0.0, 255.0) as u8;
                        self.buffer[idx] = blend_pixel(self.buffer[idx], color, alpha);
                    }
                } else {
                    self.buffer[idx] = color;
                }
            }
        }
    }

    fn draw_rect_outline(&mut self, x: i16, y: i16, w: u16, h: u16, color: u32) {
        self.fill_rect(x, y, w, 1, color);
        self.fill_rect(x, y + h as i16 - 1, w, 1, color);
        self.fill_rect(x, y, 1, h, color);
        self.fill_rect(x + w as i16 - 1, y, 1, h, color);
    }

    fn draw_rounded_outline(&mut self, x: i16, y: i16, w: u16, h: u16, r: i16, color: u32) {
        let r = r.min((w / 2) as i16).min((h / 2) as i16).max(0);
        let r_f = r as f32;
        let stroke_w = 1.0f32;

        for dy in 0..(h as i16) {
            let py = y + dy;
            if py < 0 || py >= self.height as i16 { continue; }
            let row_offset = (py as usize) * self.width;

            for dx in 0..(w as i16) {
                let px = x + dx;
                if px < 0 || px >= self.width as i16 { continue; }

                let (cx, cy) = if dx < r && dy < r {
                    (r_f - (dx as f32 + 0.5), r_f - (dy as f32 + 0.5))
                } else if dx >= (w as i16 - r) && dy < r {
                    ((dx as f32 + 0.5) - (w as f32 - r_f), r_f - (dy as f32 + 0.5))
                } else if dx < r && dy >= (h as i16 - r) {
                    (r_f - (dx as f32 + 0.5), (dy as f32 + 0.5) - (h as f32 - r_f))
                } else if dx >= (w as i16 - r) && dy >= (h as i16 - r) {
                    ((dx as f32 + 0.5) - (w as f32 - r_f), (dy as f32 + 0.5) - (h as f32 - r_f))
                } else {
                    (-1.0, -1.0)
                };

                let idx = row_offset + px as usize;
                if cx >= 0.0 {
                    let dist = (cx * cx + cy * cy).sqrt();
                    let diff = (dist - (r_f - stroke_w / 2.0)).abs();
                    if diff < 0.8 {
                        let alpha = ((1.0 - (diff / 0.8)) * 255.0).clamp(0.0, 255.0) as u8;
                        self.buffer[idx] = blend_pixel(self.buffer[idx], color, alpha);
                    }
                } else {
                    if dx == 0 || dx == (w as i16 - 1) || dy == 0 || dy == (h as i16 - 1) {
                        self.buffer[idx] = color;
                    }
                }
            }
        }
    }

    fn fill_circle_aa(&mut self, cx: i16, cy: i16, r: f32, color: u32) {
        let min_x = (cx as f32 - r - 1.0).floor() as i16;
        let max_x = (cx as f32 + r + 1.0).ceil() as i16;
        let min_y = (cy as f32 - r - 1.0).floor() as i16;
        let max_y = (cy as f32 + r + 1.0).ceil() as i16;

        for py in min_y..=max_y {
            if py < 0 || py >= self.height as i16 { continue; }
            let dy = py as f32 + 0.5 - cy as f32;
            let row_offset = (py as usize) * self.width;

            for px in min_x..=max_x {
                if px < 0 || px >= self.width as i16 { continue; }
                let dx = px as f32 + 0.5 - cx as f32;
                let dist = (dx * dx + dy * dy).sqrt();

                if dist <= r - 0.5 {
                    self.buffer[row_offset + px as usize] = color;
                } else if dist < r + 0.5 {
                    let alpha = ((r + 0.5 - dist) * 255.0).clamp(0.0, 255.0) as u8;
                    let idx = row_offset + px as usize;
                    self.buffer[idx] = blend_pixel(self.buffer[idx], color, alpha);
                }
            }
        }
    }

    fn draw_circle_outline_aa(&mut self, cx: i16, cy: i16, r: f32, stroke: f32, color: u32) {
        let half = stroke / 2.0;
        let min_x = (cx as f32 - r - half - 1.0).floor() as i16;
        let max_x = (cx as f32 + r + half + 1.0).ceil() as i16;
        let min_y = (cy as f32 - r - half - 1.0).floor() as i16;
        let max_y = (cy as f32 + r + half + 1.0).ceil() as i16;

        for py in min_y..=max_y {
            if py < 0 || py >= self.height as i16 { continue; }
            let dy = py as f32 + 0.5 - cy as f32;
            let row_offset = (py as usize) * self.width;

            for px in min_x..=max_x {
                if px < 0 || px >= self.width as i16 { continue; }
                let dx = px as f32 + 0.5 - cx as f32;
                let dist = (dx * dx + dy * dy).sqrt();
                let diff = (dist - r).abs();

                if diff <= half - 0.5 {
                    self.buffer[row_offset + px as usize] = color;
                } else if diff < half + 0.5 {
                    let alpha = ((half + 0.5 - diff) * 255.0).clamp(0.0, 255.0) as u8;
                    let idx = row_offset + px as usize;
                    self.buffer[idx] = blend_pixel(self.buffer[idx], color, alpha);
                }
            }
        }
    }

    fn draw_line_aa(&mut self, x0: f32, y0: f32, x1: f32, y1: f32, stroke: f32, color: u32) {
        let half = stroke / 2.0;
        let min_x = (x0.min(x1) - half - 1.0).floor() as i16;
        let max_x = (x0.max(x1) + half + 1.0).ceil() as i16;
        let min_y = (y0.min(y1) - half - 1.0).floor() as i16;
        let max_y = (y0.max(y1) + half + 1.0).ceil() as i16;

        let dx = x1 - x0;
        let dy = y1 - y0;
        let len_sq = dx * dx + dy * dy;
        if len_sq < 0.0001 {
            self.fill_circle_aa(x0.round() as i16, y0.round() as i16, half, color);
            return;
        }

        for py in min_y..=max_y {
            if py < 0 || py >= self.height as i16 { continue; }
            let row_offset = (py as usize) * self.width;
            let p_y = py as f32 + 0.5;

            for px in min_x..=max_x {
                if px < 0 || px >= self.width as i16 { continue; }
                let p_x = px as f32 + 0.5;

                let t = (((p_x - x0) * dx + (p_y - y0) * dy) / len_sq).clamp(0.0, 1.0);
                let proj_x = x0 + t * dx;
                let proj_y = y0 + t * dy;

                let dist = ((p_x - proj_x) * (p_x - proj_x) + (p_y - proj_y) * (p_y - proj_y)).sqrt();
                if dist <= half - 0.5 {
                    self.buffer[row_offset + px as usize] = color;
                } else if dist < half + 0.5 {
                    let alpha = ((half + 0.5 - dist) * 255.0).clamp(0.0, 255.0) as u8;
                    let idx = row_offset + px as usize;
                    self.buffer[idx] = blend_pixel(self.buffer[idx], color, alpha);
                }
            }
        }
    }

    fn fill_triangle_aa(&mut self, p0: (f32, f32), p1: (f32, f32), p2: (f32, f32), color: u32) {
        let min_x = (p0.0.min(p1.0).min(p2.0) - 1.0).floor() as i16;
        let max_x = (p0.0.max(p1.0).max(p2.0) + 1.0).ceil() as i16;
        let min_y = (p0.1.min(p1.1).min(p2.1) - 1.0).floor() as i16;
        let max_y = (p0.1.max(p1.1).max(p2.1) + 1.0).ceil() as i16;

        let edge = |a: (f32, f32), b: (f32, f32), c: (f32, f32)| -> f32 {
            (c.0 - a.0) * (b.1 - a.1) - (c.1 - a.1) * (b.0 - a.0)
        };

        let a = p0;
        let mut b = p1;
        let mut c = p2;
        let area = edge(a, b, c);
        if area < 0.0 {
            std::mem::swap(&mut b, &mut c);
        }
        if edge(a, b, c).abs() < 0.001 { return; }

        let e0_len = ((b.0 - a.0).powi(2) + (b.1 - a.1).powi(2)).sqrt().max(0.001);
        let e1_len = ((c.0 - b.0).powi(2) + (c.1 - b.1).powi(2)).sqrt().max(0.001);
        let e2_len = ((a.0 - c.0).powi(2) + (a.1 - c.1).powi(2)).sqrt().max(0.001);

        for py in min_y..=max_y {
            if py < 0 || py >= self.height as i16 { continue; }
            let row_offset = (py as usize) * self.width;
            let p_y = py as f32 + 0.5;

            for px in min_x..=max_x {
                if px < 0 || px >= self.width as i16 { continue; }
                let p_x = px as f32 + 0.5;
                let pt = (p_x, p_y);

                let d0 = edge(a, b, pt) / e0_len;
                let d1 = edge(b, c, pt) / e1_len;
                let d2 = edge(c, a, pt) / e2_len;

                let min_d = d0.min(d1).min(d2);
                if min_d >= 0.5 {
                    self.buffer[row_offset + px as usize] = color;
                } else if min_d > -0.5 {
                    let alpha = ((min_d + 0.5) * 255.0).clamp(0.0, 255.0) as u8;
                    let idx = row_offset + px as usize;
                    self.buffer[idx] = blend_pixel(self.buffer[idx], color, alpha);
                }
            }
        }
    }

    fn fill_pill(&mut self, x: i16, y: i16, w: u16, h: u16, color: u32) {
        let r = (h / 2) as i16;
        self.fill_rounded_rect(x, y, w, h, r, color);
    }

    fn draw_button_with_icon(&mut self, x: i16, y: i16, w: u16, h: u16, icon_id: &str, bg: u32, fg: u32, id: &str) {
        let r = ((h as i16) / 2).min(12);
        self.fill_rounded_rect(x, y, w, h, r, bg);
        self.draw_rounded_outline(x, y, w, h, r, blend_pixel(bg, M3_OUTLINE_VARIANT, 60));

        let icon_cx = x + (w as i16) / 2;
        let icon_cy = y + (h as i16) / 2;
        draw_vector_icon(self, icon_cx, icon_cy, icon_id, fg, (w.min(h) as f32 * 0.5) as u16);
        self.register_button(x, y, w, h, id);
    }

    fn split_script_runs<'b>(&self, text: &'b str) -> Vec<(&'b str, bool)> {
        let mut runs = Vec::new();
        let mut chars = text.char_indices().peekable();

        while let Some((start_idx, ch)) = chars.next() {
            let is_bengali = is_bengali_char(ch);
            let mut end_idx = text.len();

            while let Some(&(next_idx, next_ch)) = chars.peek() {
                let next_is_bengali = is_bengali_char(next_ch);
                if next_is_bengali != is_bengali {
                    end_idx = next_idx;
                    break;
                }
                chars.next();
            }

            runs.push((&text[start_idx..end_idx], is_bengali));
        }

        runs
    }

    fn draw_text_smooth(&mut self, x: i16, y: i16, px_size: f32, text: &str, color: u32, is_mono: bool) -> i16 {
        if text.is_empty() {
            return x;
        }

        let runs = self.split_script_runs(text);
        let mut cur_x = x;

        for (run_text, is_bengali) in runs {
            if is_bengali {
                if let Some(face) = rustybuzz::Face::from_slice(&self.fonts.bengali_face_bytes, 0) {
                    let mut buffer = rustybuzz::UnicodeBuffer::new();
                    buffer.push_str(run_text);
                    buffer.guess_segment_properties();

                    let glyph_buffer = rustybuzz::shape(&face, &[], buffer);
                    let upem = face.units_per_em() as f32;
                    let scale = px_size / upem;

                    for (info, pos) in glyph_buffer.glyph_infos().iter().zip(glyph_buffer.glyph_positions().iter()) {
                        let glyph_id = info.glyph_id as u16;
                        let (metrics, bitmap) = self.fonts.bengali_font.rasterize_indexed(glyph_id, px_size);

                        let x_offset = (pos.x_offset as f32 * scale).round() as i16;
                        let y_offset = (pos.y_offset as f32 * scale).round() as i16;

                        let char_x = cur_x + x_offset + metrics.xmin as i16;
                        let char_y = y + (px_size as i16 - metrics.ymin as i16 - metrics.height as i16) - y_offset;

                        for (row, dy) in (0..metrics.height).enumerate() {
                            let py = char_y + dy as i16;
                            if py < 0 || py >= self.height as i16 {
                                continue;
                            }
                            let row_offset = (py as usize) * self.width;
                            for (col, dx) in (0..metrics.width).enumerate() {
                                let px = char_x + dx as i16;
                                if px >= 0 && px < (self.width as i16 - 4) {
                                    let alpha = bitmap[row * metrics.width + col];
                                    if alpha > 0 {
                                        let idx = row_offset + px as usize;
                                        self.buffer[idx] = blend_pixel(self.buffer[idx], color, alpha);
                                    }
                                }
                            }
                        }
                        cur_x += (pos.x_advance as f32 * scale).round() as i16;
                    }
                }
            } else {
                let font = if is_mono { &self.fonts.mono_font } else { &self.fonts.ui_font };
                for ch in run_text.chars() {
                    if ch == '\n' {
                        break;
                    }
                    let (metrics, bitmap) = font.rasterize(ch, px_size);
                    let char_x = cur_x + metrics.xmin as i16;
                    let char_y = y + (px_size as i16 - metrics.ymin as i16 - metrics.height as i16);

                    for (row, dy) in (0..metrics.height).enumerate() {
                        let py = char_y + dy as i16;
                        if py < 0 || py >= self.height as i16 {
                            continue;
                        }
                        let row_offset = (py as usize) * self.width;
                        for (col, dx) in (0..metrics.width).enumerate() {
                            let px = char_x + dx as i16;
                            if px >= 0 && px < (self.width as i16 - 4) {
                                let alpha = bitmap[row * metrics.width + col];
                                if alpha > 0 {
                                    let idx = row_offset + px as usize;
                                    self.buffer[idx] = blend_pixel(self.buffer[idx], color, alpha);
                                }
                            }
                        }
                    }
                    cur_x += (metrics.advance_width.round() as i16).max(1);
                }
            }
        }

        cur_x
    }

    fn text_width(&self, px_size: f32, text: &str, is_mono: bool) -> i16 {
        if text.is_empty() {
            return 0;
        }

        let runs = self.split_script_runs(text);
        let mut total_w: f32 = 0.0;

        for (run_text, is_bengali) in runs {
            if is_bengali {
                if let Some(face) = rustybuzz::Face::from_slice(&self.fonts.bengali_face_bytes, 0) {
                    let mut buffer = rustybuzz::UnicodeBuffer::new();
                    buffer.push_str(run_text);
                    buffer.guess_segment_properties();
                    let glyph_buffer = rustybuzz::shape(&face, &[], buffer);
                    let upem = face.units_per_em() as f32;
                    let scale = px_size / upem;
                    let w: f32 = glyph_buffer.glyph_positions().iter().map(|p| p.x_advance as f32 * scale).sum();
                    total_w += w;
                }
            } else {
                let font = if is_mono { &self.fonts.mono_font } else { &self.fonts.ui_font };
                for ch in run_text.chars() {
                    let metrics = font.metrics(ch, px_size);
                    total_w += metrics.advance_width;
                }
            }
        }

        total_w.round() as i16
    }

    fn register_button(&mut self, x: i16, y: i16, w: u16, h: u16, id: &str) {
        self.buttons.push(TouchButton {
            x,
            y,
            w,
            h,
            id: id.to_string(),
        });
    }

    fn draw_button(&mut self, x: i16, y: i16, w: u16, h: u16, label: &str, bg: u32, fg: u32, id: &str) {
        let r = ((h as i16) / 2).min(12);
        self.fill_rounded_rect(x, y, w, h, r, bg);
        self.draw_rounded_outline(x, y, w, h, r, blend_pixel(bg, M3_OUTLINE_VARIANT, 60));

        let font_size = if h < 28 { 12.0 } else if h < 36 { 13.5 } else { 14.5 };
        let tw = self.text_width(font_size, label, false);
        let text_x = x + ((w as i16 - tw) / 2).max(3);
        let text_y = y + ((h as i16 - font_size as i16) / 2).max(2);

        self.draw_text_smooth(text_x, text_y, font_size, label, fg, false);
        self.register_button(x, y, w, h, id);
    }

    fn draw_m3_chip(&mut self, x: i16, y: i16, text: &str, active: bool, id: &str) -> i16 {
        let tw = self.text_width(11.5, text, false);
        let chip_w = (tw + 20).max(48) as u16;
        let chip_h = 28u16;

        let (bg, border, fg) = if active {
            (M3_SECONDARY_CONTAINER, M3_SECONDARY, M3_ON_SECONDARY_CONTAINER)
        } else {
            (M3_SURFACE_CONTAINER_LOW, M3_OUTLINE_VARIANT, M3_ON_SURFACE_VARIANT)
        };

        self.fill_pill(x, y, chip_w, chip_h, bg);
        self.draw_rounded_outline(x, y, chip_w, chip_h, (chip_h / 2) as i16, border);
        self.draw_text_smooth(x + 10, y + 6, 11.5, text, fg, false);
        self.register_button(x, y, chip_w, chip_h, id);

        chip_w as i16
    }

    fn draw_m3_fab(&mut self, x: i16, y: i16, label: &str, id: &str) {
        let size = 52u16;
        let r = 16i16;
        self.fill_rounded_rect(x, y, size, size, r, M3_PRIMARY_CONTAINER);
        self.draw_rounded_outline(x, y, size, size, r, M3_PRIMARY);

        let tw = self.text_width(18.0, label, false);
        self.draw_text_smooth(x + (size as i16 - tw) / 2, y + 16, 18.0, label, M3_ON_PRIMARY_CONTAINER, false);
        self.register_button(x, y, size, size, id);
    }

    fn draw_app_icon(&mut self, x: i16, y: i16, symbol: &str, label: &str, icon_bg: u32, symbol_color: u32, id: &str) {
        let icon_size: i16 = 46;
        let r = 12;

        // Outer glow/shadow (softer)
        let glow = blend_pixel(COLOR_BG, icon_bg, 28);
        self.fill_rounded_rect(x + 7, y - 1, (icon_size + 2) as u16, (icon_size + 2) as u16, r + 1, glow);

        // Icon body
        self.fill_rounded_rect(x + 8, y, icon_size as u16, icon_size as u16, r, icon_bg);

        // Glassy sheen (thin top highlight)
        let sheen = blend_pixel(icon_bg, 0xFFFFFF, 32);
        self.fill_rounded_rect(x + 10, y + 2, (icon_size - 4) as u16, 12, r - 2, sheen);

        // Symbol centered
        let sym_size = 18.0;
        let sw = self.text_width(sym_size, symbol, false);
        let sym_x = (x + 8) + ((icon_size - sw) / 2);
        let sym_y = y + ((icon_size - sym_size as i16) / 2) + 1;
        self.draw_text_smooth(sym_x, sym_y, sym_size, symbol, symbol_color, false);

        // Label below icon (smaller, tighter)
        let lbl_size = 10.5;
        let lw = self.text_width(lbl_size, label, false);
        let cell_w = 62i16;
        let lbl_x = x + ((cell_w - lw) / 2).max(0);
        let lbl_y = y + icon_size + 5;
        self.draw_text_smooth(lbl_x, lbl_y, lbl_size, label, COLOR_TEXT_HIGH, false);

        self.register_button(x, y, 62, (icon_size + 22) as u16, id);
    }

    fn draw_pill_badge(&mut self, x: i16, y: i16, text: &str, bg: u32, fg: u32) {
        let tw = self.text_width(11.0, text, false);
        let pw = (tw + 14).max(24) as u16;
        self.fill_rounded_rect(x, y, pw, 18, 9, bg);
        self.draw_text_smooth(x + 7, y + 3, 11.0, text, fg, false);
    }
}

// ─── Word Wrapping Helper ─────────────────────────────────────────────────────
fn wrap_text_to_lines(painter: &FramePainter, text: &str, max_width: i16, font_size: f32) -> Vec<String> {
    let mut wrapped = Vec::new();
    for raw_line in text.split('\n') {
        if raw_line.is_empty() {
            wrapped.push(String::new());
            continue;
        }

        let words: Vec<&str> = raw_line.split(' ').collect();
        let mut cur_line = String::new();

        for word in words {
            let test_line = if cur_line.is_empty() {
                word.to_string()
            } else {
                format!("{} {}", cur_line, word)
            };

            if painter.text_width(font_size, &test_line, false) > max_width {
                if !cur_line.is_empty() {
                    wrapped.push(cur_line);
                    cur_line = word.to_string();
                } else {
                    wrapped.push(test_line);
                    cur_line.clear();
                }
            } else {
                cur_line = test_line;
            }
        }

        if !cur_line.is_empty() {
            wrapped.push(cur_line);
        }
    }
    wrapped
}

fn safe_truncate(s: &str, max_chars: usize) -> String {
    let chars: Vec<char> = s.chars().collect();
    if chars.len() > max_chars {
        let truncated: String = chars[..max_chars].iter().collect();
        format!("{}...", truncated)
    } else {
        s.to_string()
    }
}

fn safe_truncate_end(s: &str, max_chars: usize) -> String {
    let chars: Vec<char> = s.chars().collect();
    if chars.len() > max_chars {
        let start = chars.len() - max_chars;
        let suffix: String = chars[start..].iter().collect();
        format!("...{}", suffix)
    } else {
        s.to_string()
    }
}

fn extract_html_title(html: &str) -> String {
    let lower = html.to_lowercase();
    if let Some(start_tag) = lower.find("<title>") {
        let start = start_tag + 7;
        if let Some(end_tag) = lower[start..].find("</title>") {
            let end = start + end_tag;
            if let Some(extracted) = html.get(start..end) {
                let clean = extracted.trim().to_string();
                if !clean.is_empty() {
                    return clean;
                }
            }
        }
    }
    "ওয়েব পেজ (Live Web)".to_string()
}

fn decode_html_entities(s: &str) -> String {
    s.replace("&amp;", "&")
     .replace("&quot;", "\"")
     .replace("&#39;", "'")
     .replace("&apos;", "'")
     .replace("&lt;", "<")
     .replace("&gt;", ">")
     .replace("&nbsp;", " ")
     .replace("&#x27;", "'")
     .replace("&#x2F;", "/")
     .replace("&#8217;", "'")
     .replace("&#8216;", "'")
     .replace("&#8220;", "\"")
     .replace("&#8221;", "\"")
     .replace("&#8211;", "-")
     .replace("&#8212;", "-")
}

fn clean_html_to_text(html: &str) -> String {
    let mut clean = String::new();
    let mut in_tag = false;
    let mut in_script = false;
    let mut in_style = false;
    let chars: Vec<char> = html.chars().collect();
    let len = chars.len();
    let mut i = 0;

    while i < len {
        if !in_tag && i + 7 < len {
            let slice: String = chars[i..i+7].iter().collect();
            if slice.to_lowercase() == "<script" {
                in_script = true;
                i += 7;
                continue;
            }
        }
        if in_script {
            if i + 9 < len {
                let slice: String = chars[i..i+9].iter().collect();
                if slice.to_lowercase() == "</script>" {
                    in_script = false;
                    i += 9;
                    continue;
                }
            }
            i += 1;
            continue;
        }

        if !in_tag && i + 6 < len {
            let slice: String = chars[i..i+6].iter().collect();
            if slice.to_lowercase() == "<style" {
                in_style = true;
                i += 6;
                continue;
            }
        }
        if in_style {
            if i + 8 < len {
                let slice: String = chars[i..i+8].iter().collect();
                if slice.to_lowercase() == "</style>" {
                    in_style = false;
                    i += 8;
                    continue;
                }
            }
            i += 1;
            continue;
        }

        let c = chars[i];
        if c == '<' {
            in_tag = true;
        } else if c == '>' {
            in_tag = false;
            clean.push('\n');
        } else if !in_tag {
            clean.push(c);
        }
        i += 1;
    }
    decode_html_entities(&clean)
}

// ─── Data Structures ──────────────────────────────────────────────────────────
#[derive(Clone, Debug)]
struct TermLine {
    text: String,
    color: u32,
}

#[derive(Clone, Debug)]
struct Note {
    id: usize,
    title: String,
    content: String,
    category: String,
    updated: String,
    pinned: bool,
    color: u32,
}

#[derive(Clone, Debug)]
struct WebLine {
    text: String,
    is_heading: bool,
    color: u32,
}

#[derive(Clone, Debug)]
struct HistoryEntry {
    url: String,
    title: String,
    time_str: String,
}

#[derive(Clone, Debug)]
struct Bookmark {
    id: String,
    name: String,
    url: String,
    icon: String,
}

#[derive(Clone, Debug)]
struct BrowserTab {
    id: usize,
    title: String,
    url: String,
}

#[derive(Clone, Debug)]
struct MusicTrack {
    title: &'static str,
    artist: &'static str,
    duration: &'static str,
    duration_secs: u32,
    color: u32,
}

const MUSIC_TRACKS: &[MusicTrack] = &[
    MusicTrack {
        title: "আনন্দলোকে মঙ্গলালোকে",
        artist: "রবীন্দ্রনাথ ঠাকুর — ভক্তিমূলক",
        duration: "০৩:৪৫",
        duration_secs: 225,
        color: COLOR_PURPLE,
    },
    MusicTrack {
        title: "অনুরণ ওএস সিম্ফনি নং ১",
        artist: "নীল কোর টিম — আল্ট্রা সাউন্ড",
        duration: "০৪:১২",
        duration_secs: 252,
        color: COLOR_CYAN,
    },
    MusicTrack {
        title: "লো-ফাই কোডিং বিটস",
        artist: "NilBeats Studio — লো-ফাই",
        duration: "০২:৫৮",
        duration_secs: 178,
        color: COLOR_GREEN,
    },
    MusicTrack {
        title: "কারার ঐ লৌহকপাট",
        artist: "কাজী নজরুল ইসলাম — বিদ্রোহী",
        duration: "০৩:১৫",
        duration_secs: 195,
        color: COLOR_AMBER,
    },
];

// ─── Chat / Messaging Types ──────────────────────────────────────────────────
#[derive(Clone, Debug)]
struct ChatMessageItem {
    sender: String,
    text: String,
    time: String,
    is_me: bool,
}

#[derive(Clone, Debug)]
struct ChatThreadItem {
    name: String,
    avatar_char: String,
    messages: Vec<ChatMessageItem>,
}

// ─── Application State ────────────────────────────────────────────────────────
struct SimState {
    screen: Screen,
    home_page: usize,
    user_name: String,
    pin: String,
    pin_input: String,
    lock_error: bool,
    dial_number: String,
    phone_calling: bool,
    phone_call_duration: u32,
    phone_tab: usize, // 0 = ডায়ালার, 1 = সাম্প্রতিক কল, 2 = পরিচিতি
    phone_recents: Vec<(String, String, String, bool)>, // (name_or_number, type, time, is_missed)
    phone_contacts: Vec<(String, String, String)>,       // (name, number, role)
    sms_threads: Vec<ChatThreadItem>,
    sms_active_thread: Option<usize>,
    sms_reply_input: String,
    sms_auto_reply_ticks: Option<(usize, usize, String)>,
    current_path: String,
    installed_pkgs: Vec<String>,
    wifi_enabled: bool,
    wifi_signal_level: u8,
    cellular_enabled: bool,
    cellular_signal_level: u8,
    battery_level: u8,
    battery_charging: bool,
    bt_enabled: bool,
    softbus_enabled: bool,
    dark_mode: bool,
    torch_enabled: bool,
    airplane_mode: bool,
    hotspot_enabled: bool,
    brightness_level: u8,
    volume_level: u8,

    // Real Linux Terminal State
    storage_root: PathBuf,
    term_input: String,
    term_cursor_pos: usize,
    term_cwd: PathBuf,
    term_lines: Vec<TermLine>,
    term_history: Vec<String>,
    term_history_idx: Option<usize>,
    term_scroll_offset: usize,
    term_cursor_ticks: usize,
    python_mode: bool,
    bash_path: Option<String>,

    // Nano Text Editor State
    nano_filename: String,
    nano_lines: Vec<String>,
    nano_row: usize,
    nano_col: usize,
    nano_dirty: bool,
    nano_status: String,

    // ArkTS Notes App State
    notes: Vec<Note>,
    notes_category: String,
    notes_editing: bool,
    notes_edit_id: Option<usize>,
    notes_edit_title: String,
    notes_edit_content: String,
    notes_cursor_pos: usize,

    // ArkTS Calculator State
    calc_expr: String,
    calc_result: String,
    calc_history: Vec<(String, String)>,

    // ArkTS Music Player State
    music_playing: bool,
    music_track_idx: usize,
    music_progress: f32,
    music_current_sec: u32,
    music_tick: usize,

    // NilZar Browser State
    browser_url: String,
    browser_url_input: String,
    browser_is_editing_url: bool,
    browser_cursor_pos: usize,
    browser_title: String,
    browser_status: String,
    browser_loading: bool,
    browser_progress: f32,
    browser_show_history: bool,
    browser_show_bookmarks: bool,
    browser_history: Vec<HistoryEntry>,
    browser_bookmarks: Vec<Bookmark>,
    browser_web_lines: Vec<WebLine>,
    browser_scroll_offset: usize,

    browser_tabs: Vec<BrowserTab>,
    browser_active_tab: usize,
    browser_next_tab_id: usize,
    browser_tab_scroll: i16,
}

impl SimState {
    fn new() -> Self {
        let (storage_root, home_dir) = ensure_nilos_storage();
        let clean_cwd = home_dir.clone();

        let bash_path = if Path::new(r"C:\Program Files\Git\bin\bash.exe").exists() {
            Some(r"C:\Program Files\Git\bin\bash.exe".to_string())
        } else if Path::new(r"C:\Program Files\Git\usr\bin\bash.exe").exists() {
            Some(r"C:\Program Files\Git\usr\bin\bash.exe".to_string())
        } else {
            None
        };

        let mut lines = Vec::new();
        lines.push(TermLine {
            text: "================================================".into(),
            color: COLOR_CYAN,
        });
        lines.push(TermLine {
            text: "  অনুরণ ওএস লিনাক্স ব্যাশ টার্মিনাল (Onuron GNU Bash)  ".into(),
            color: COLOR_CYAN,
        });
        lines.push(TermLine {
            text: "  কীবোর্ড তীর চিহ্ন (← / →) দিয়ে এডিট করুন।       ".into(),
            color: COLOR_TEXT_MED,
        });
        lines.push(TermLine {
            text: "================================================".into(),
            color: COLOR_BORDER,
        });
        lines.push(TermLine {
            text: "কমান্ড: nil run <file.nil>, nil hir, nil disasm, nilpos, vlc, ls".into(),
            color: COLOR_AMBER,
        });

        let default_notes = vec![
            Note {
                id: 1,
                title: "নীল ওএস কার্নেল আর্কিটেকচার".into(),
                content: "মাইক্রোকার্নেল বেসড IPC ও SoftBus ডিস্ট্রিবিউটেড ফ্যাব্রিক প্রস্তুত।\n- zImage বুট ও Ramdisk লোড\n- fscrypt v2 এনক্রিপশন।".into(),
                category: "কাজের নোট".into(),
                updated: "১৬:৩০".into(),
                pinned: true,
                color: COLOR_CYAN,
            },
            Note {
                id: 2,
                title: "ArkTS অ্যাপ ডেভেলপমেন্ট গাইড".into(),
                content: "ডিক্লেয়ারেটিভ UI দিয়ে দ্রুত ও স্মুথ অ্যাপ তৈরি করা সম্ভব। ArkCompiler সরাসরি বাইটকোড অপ্টিমাইজ করে।".into(),
                category: "কোড ও আইডিয়া".into(),
                updated: "১৬:১৫".into(),
                pinned: true,
                color: COLOR_GREEN,
            },
            Note {
                id: 3,
                title: "বাজারের তালিকা (Shopping)".into(),
                content: "১. ল্যাপটপ স্ট্যান্ড\n২. টাইপ-সি OTG কেবল\n৩. কফি বিন্স।".into(),
                category: "ব্যক্তিগত".into(),
                updated: "১৫:৫০".into(),
                pinned: false,
                color: COLOR_AMBER,
            },
        ];

        let initial_web_lines = vec![
            WebLine {
                text: "Google".into(),
                is_heading: true,
                color: COLOR_CYAN,
            },
            WebLine {
                text: "বিশ্বের বৃহত্তম তথ্যভাণ্ডার ও সার্চ ইঞ্জিন।".into(),
                is_heading: false,
                color: COLOR_TEXT_MED,
            },
            WebLine {
                text: "🔍 যেকোনো তথ্য বা প্রশ্ন লিখে উপরের অ্যাড্রেস বারে 'গো' চাপুন।".into(),
                is_heading: true,
                color: COLOR_GREEN,
            },
            WebLine {
                text: "──────── ট্রেন্ডিং ও খবর ────────".into(),
                is_heading: false,
                color: COLOR_BORDER,
            },
            WebLine {
                text: "• অনুরণ ওএস (Onuron OS) বাংলা লিনাক্স অপারেটিং সিস্টেম সংস্করণ ১.৭ প্রকাশ।".into(),
                is_heading: false,
                color: COLOR_TEXT_HIGH,
            },
            WebLine {
                text: "• SoftBus ডিস্ট্রিবিউটেড পিয়ার-টু-পিয়ার ফ্যাব্রিক নেটওয়ার্ক সক্রিয়।".into(),
                is_heading: false,
                color: COLOR_TEXT_HIGH,
            },
            WebLine {
                text: "• ভারতীয় সময় (IST UTC+5:30) অনুযায়ী সম্পূর্ণ বাংলা ক্যালেন্ডার।".into(),
                is_heading: false,
                color: COLOR_TEXT_HIGH,
            },
            WebLine {
                text: "───────────────────────────────".into(),
                is_heading: false,
                color: COLOR_BORDER,
            },
            WebLine {
                text: "Google offered in: বাংলা (ভারত) English हिन्दी".into(),
                is_heading: false,
                color: COLOR_AMBER,
            },
        ];

        let default_bookmarks = vec![
            Bookmark { id: "1".into(), name: "Onuron GitHub".into(), url: "https://github.com/joysriramsarkar/nilos".into(), icon: "🏠".into() },
            Bookmark { id: "2".into(), name: "Google".into(), url: "https://google.com".into(), icon: "🌐".into() },
            Bookmark { id: "3".into(), name: "উইকিপিডিয়া".into(), url: "https://bn.wikipedia.org".into(), icon: "📚".into() },
            Bookmark { id: "4".into(), name: "DuckDuckGo".into(), url: "https://duckduckgo.com".into(), icon: "🦆".into() },
            Bookmark { id: "5".into(), name: "Go Language".into(), url: "https://go.dev".into(), icon: "⚡".into() },
        ];

        let default_history = vec![
            HistoryEntry { url: "https://google.com".into(), title: "Google Search".into(), time_str: "১৭:২৫".into() },
            HistoryEntry { url: "https://bn.wikipedia.org".into(), title: "উইকিপিডিয়া — মুক্ত বিশ্বকোষ".into(), time_str: "১৭:২০".into() },
            HistoryEntry { url: "https://onuron.dev".into(), title: "Onuron OS Official Portal".into(), time_str: "১৭:১৫".into() },
        ];

        SimState {
            screen: Screen::Home,
            home_page: 0,
            user_name: "জয় সরকার".into(),
            pin: "1234".into(),
            pin_input: String::new(),
            lock_error: false,
            dial_number: String::new(),
            phone_calling: false,
            phone_call_duration: 0,
            phone_tab: 0,
            phone_recents: vec![
                ("জয় সরকার".into(), "আউটগোয়িং • ০৩:২৪".into(), "১০:১৫".into(), false),
                ("অনুরণ হেল্পলাইন".into(), "ইনকামিং • ০১:১২".into(), "গতকাল".into(), false),
                ("অজানা নম্বর (+91 98320...)".into(), "মিসড কল".into(), "গতকাল".into(), true),
                ("অনির্বাণ সেন".into(), "আউটগোয়িং • ০২:৪৫".into(), "৩ দিন আগে".into(), false),
            ],
            phone_contacts: vec![
                ("জয় সরকার".into(), "+91 98765 43210".into(), "নিজের নম্বর".into()),
                ("অনির্বাণ সেন".into(), "+91 98321 55443".into(), "ডেভেলপার".into()),
                ("অনুরণ সাপোর্ট".into(), "+91 80000 12345".into(), "সিস্টেম সহায়তা".into()),
                ("মা".into(), "+91 94340 98765".into(), "পরিবার".into()),
                ("অভিজিৎ রায়".into(), "+91 97330 11223".into(), "বন্ধু".into()),
            ],
            sms_threads: vec![
                ChatThreadItem {
                    name: "জয় সরকার".into(),
                    avatar_char: "জ".into(),
                    messages: vec![
                        ChatMessageItem { sender: "জয় সরকার".into(), text: "অনুরণ ওএস আর্কিটেকচার খুব দারুণ হচ্ছে!".into(), time: "১০:৪০".into(), is_me: false },
                        ChatMessageItem { sender: "আমি".into(), text: "ধন্যবাদ! নতুন ভেক্টর গ্রাফিক্স আর অডিও কাজ করছে।".into(), time: "১০:৪২".into(), is_me: true },
                        ChatMessageItem { sender: "জয় সরকার".into(), text: "নতুন মেসেজিং অ্যাপ চালিয়ে দেখুন।".into(), time: "১০:৪৩".into(), is_me: false },
                    ],
                },
                ChatThreadItem {
                    name: "সফটবাস মেশ".into(),
                    avatar_char: "স".into(),
                    messages: vec![
                        ChatMessageItem { sender: "সফটবাস মেশ".into(), text: "NilPad-Pro-X1 সফলভাবে যুক্ত হয়েছে।".into(), time: "০৯:১৫".into(), is_me: false },
                        ChatMessageItem { sender: "আমি".into(), text: "P2P QUIC কানেকশন ঠিকঠাক আছে?".into(), time: "০৯:১৬".into(), is_me: true },
                        ChatMessageItem { sender: "সফটবাস মেশ".into(), text: "হ্যাঁ, 0ms লেটেন্সি এবং 128-bit এনক্রিপশন সক্রিয়।".into(), time: "০৯:১৭".into(), is_me: false },
                    ],
                },
                ChatThreadItem {
                    name: "নীল ওএস সিস্টেম".into(),
                    avatar_char: "ন".into(),
                    messages: vec![
                        ChatMessageItem { sender: "নীল ওএস সিস্টেম".into(), text: "ফাইলসিস্টেম v2 স্টোরেজ এনক্রিপশন সক্রিয়।".into(), time: "০৮:০০".into(), is_me: false },
                        ChatMessageItem { sender: "নীল ওএস সিস্টেম".into(), text: "সিস্টেম ফার্মওয়্যার ও HAL আপডেট প্রস্তুত।".into(), time: "০৮:০৫".into(), is_me: false },
                    ],
                },
            ],
            sms_active_thread: None,
            sms_reply_input: String::new(),
            sms_auto_reply_ticks: None,
            storage_root,
            current_path: clean_cwd.to_string_lossy().to_string(),
            installed_pkgs: vec![
                "com.nil.shell".into(),
                "com.nil.notes".into(),
                "com.nil.calc".into(),
                "com.nil.music".into(),
                "com.nil.settings".into(),
                "com.nil.softbus".into(),
                "org.mozilla.fenix".into(),
                "org.videolan.vlc".into(),
            ],
            wifi_enabled: true,
            wifi_signal_level: 3,
            cellular_enabled: true,
            cellular_signal_level: 4,
            battery_level: 88,
            battery_charging: false,
            bt_enabled: true,
            softbus_enabled: true,
            dark_mode: true,
            torch_enabled: false,
            airplane_mode: false,
            hotspot_enabled: false,
            brightness_level: 100,
            volume_level: 80,

            term_input: String::new(),
            term_cursor_pos: 0,
            term_cwd: clean_cwd,
            term_lines: lines,
            term_history: Vec::new(),
            term_history_idx: None,
            term_scroll_offset: 0,
            term_cursor_ticks: 0,
            python_mode: false,
            bash_path,

            nano_filename: "untitled.txt".into(),
            nano_lines: vec!["".into()],
            nano_row: 0,
            nano_col: 0,
            nano_dirty: false,
            nano_status: "Ctrl+O: সেভ করুন | Ctrl+X: প্রস্থান".into(),

            notes: default_notes,
            notes_category: "সব নোট".into(),
            notes_editing: false,
            notes_edit_id: None,
            notes_edit_title: String::new(),
            notes_edit_content: String::new(),
            notes_cursor_pos: 0,

            calc_expr: "০".into(),
            calc_result: "".into(),
            calc_history: vec![
                ("২৫০ × ৪".into(), "১,০০০".into()),
                ("১২৮০ + ৫১২".into(), "১,৭৯২".into()),
            ],

            music_playing: true,
            music_track_idx: 0,
            music_progress: 0.35,
            music_current_sec: 78,
            music_tick: 0,

            browser_url: "https://google.com".into(),
            browser_url_input: "https://google.com".into(),
            browser_is_editing_url: false,
            browser_cursor_pos: 18,
            browser_title: "Google".into(),
            browser_status: "✓ SSL TLS 1.3 | 200 OK | Google Mobile".into(),
            browser_loading: false,
            browser_progress: 1.0,
            browser_show_history: false,
            browser_show_bookmarks: false,
            browser_history: default_history,
            browser_bookmarks: default_bookmarks,
            browser_web_lines: initial_web_lines,
            browser_scroll_offset: 0,

            browser_tabs: vec![
                BrowserTab { id: 1, title: "Google".into(), url: "https://www.google.com".into() },
            ],
            browser_active_tab: 0,
            browser_next_tab_id: 2,
            browser_tab_scroll: 0,
        }
    }

    fn is_pkg_installed(&self, pkg_id: &str) -> bool {
        self.installed_pkgs.iter().any(|p| p == pkg_id)
    }

    fn install_pkg(&mut self, pkg_id: &str) {
        if !self.is_pkg_installed(pkg_id) {
            self.installed_pkgs.push(pkg_id.to_string());
            let name = match pkg_id {
                "org.videolan.vlc" => "ভিএলসি মিডিয়া প্লেয়ার (VLC VideoLAN)",
                "org.mozilla.fenix" => "ফায়ারফক্স প্রাইভেট ব্রাউজার",
                "com.signal.android" => "সিগন্যাল এনক্রিপ্টেড চ্যাট",
                _ => pkg_id,
            };
            self.push_term_line(format!("[+] সফলভাবে ইনস্টল সম্পন্ন: {}", name), COLOR_GREEN);
        }
    }

    fn browser_add_bookmark(&mut self) {
        let current_url = self.browser_url.clone();
        if current_url.is_empty() {
            return;
        }
        let exists = self.browser_bookmarks.iter().any(|b| b.url == current_url);
        if !exists {
            let id = format!("{}", self.browser_bookmarks.len() + 1);
            let name = self.browser_title.clone();
            self.browser_bookmarks.push(Bookmark {
                id,
                name: if name.is_empty() { current_url.clone() } else { name },
                url: current_url,
                icon: "⭐".into(),
            });
        }
        self.browser_show_bookmarks = true;
    }

    fn browser_remove_bookmark(&mut self, id: &str) {
        self.browser_bookmarks.retain(|b| b.id != id);
    }

    fn browser_fetch_live(&mut self, query_or_url: &str) {
        let clean = query_or_url.trim();
        if clean.is_empty() {
            return;
        }

        self.browser_loading = true;
        self.browser_progress = 0.3;
        self.browser_show_history = false;
        self.browser_show_bookmarks = false;

        let is_google = clean == "google.com"
            || clean == "www.google.com"
            || clean == "https://google.com"
            || clean == "http://google.com"
            || clean == "https://www.google.com"
            || clean == "http://www.google.com"
            || clean == "google"
            || clean == "গুগল";

        let is_wiki = clean == "bn.wikipedia.org"
            || clean == "wikipedia.org"
            || clean == "https://bn.wikipedia.org"
            || clean == "https://wikipedia.org"
            || clean == "উইকি"
            || clean == "উইকিপিডিয়া";

        let is_ddg = clean == "duckduckgo.com"
            || clean == "www.duckduckgo.com"
            || clean == "https://duckduckgo.com"
            || clean == "http://duckduckgo.com"
            || clean == "https://www.duckduckgo.com"
            || clean == "duckduckgo"
            || clean == "ডাকডাক"
            || clean == "ডাকডাকগো";

        if is_google {
            self.browser_url = "https://google.com".into();
            self.browser_url_input = "https://google.com".into();
            self.browser_cursor_pos = self.browser_url_input.chars().count();
            self.browser_is_editing_url = false;
            self.browser_title = "Google".into();
            self.browser_status = "✓ SSL TLS 1.3 | 200 OK | Google Mobile".into();
            self.browser_loading = false;
            self.browser_progress = 1.0;
            self.browser_scroll_offset = 0;

            self.browser_web_lines = vec![
                WebLine {
                    text: "Google".into(),
                    is_heading: true,
                    color: COLOR_CYAN,
                },
                WebLine {
                    text: "বিশ্বের বৃহত্তম তথ্যভাণ্ডার ও সার্চ ইঞ্জিন।".into(),
                    is_heading: false,
                    color: COLOR_TEXT_MED,
                },
                WebLine {
                    text: "🔍 যেকোনো কিছু লিখে উপরের অ্যাড্রেস বারে 'গো' চাপুন।".into(),
                    is_heading: true,
                    color: COLOR_GREEN,
                },
                WebLine {
                    text: "──────── ট্রেন্ডিং ও খবর ────────".into(),
                    is_heading: false,
                    color: COLOR_BORDER,
                },
                WebLine {
                    text: "• অনুরণ ওএস (Onuron OS) আধুনিক বাংলা লিনাক্স অপারেটিং সিস্টেম উন্মোচিত।".into(),
                    is_heading: false,
                    color: COLOR_TEXT_HIGH,
                },
                WebLine {
                    text: "• SoftBus ডিস্ট্রিবিউটেড পিয়ার-টু-পিয়ার মেশ নেটওয়ার্ক সক্রিয়।".into(),
                    is_heading: false,
                    color: COLOR_TEXT_HIGH,
                },
                WebLine {
                    text: "• ভারতীয় সময় (IST UTC+5:30) অনুযায়ী সম্পূর্ণ বাংলা ক্যালেন্ডার।".into(),
                    is_heading: false,
                    color: COLOR_TEXT_HIGH,
                },
                WebLine {
                    text: "───────────────────────────────".into(),
                    is_heading: false,
                    color: COLOR_BORDER,
                },
                WebLine {
                    text: "Google offered in: বাংলা (ভারত) English हिन्दी".into(),
                    is_heading: false,
                    color: COLOR_AMBER,
                },
            ];

            let entry = HistoryEntry {
                url: self.browser_url.clone(),
                title: self.browser_title.clone(),
                time_str: get_ist_time_str(),
            };
            self.browser_history.insert(0, entry);
            return;
        }

        if is_wiki {
            self.browser_url = "https://bn.wikipedia.org".into();
            self.browser_url_input = "https://bn.wikipedia.org".into();
            self.browser_cursor_pos = self.browser_url_input.chars().count();
            self.browser_is_editing_url = false;
            self.browser_title = "উইকিপিডিয়া — মুক্ত বিশ্বকোষ".into();
            self.browser_status = "✓ SSL TLS 1.3 | 200 OK | উইকিপিডিয়া".into();
            self.browser_loading = false;
            self.browser_progress = 1.0;
            self.browser_scroll_offset = 0;

            self.browser_web_lines = vec![
                WebLine {
                    text: "উইকিপিডিয়া — মুক্ত বিশ্বকোষ".into(),
                    is_heading: true,
                    color: COLOR_CYAN,
                },
                WebLine {
                    text: "বাংলা ভাষায় সবার জন্য উন্মুক্ত একটি অনলাইন জ্ঞানকোষ।".into(),
                    is_heading: false,
                    color: COLOR_TEXT_MED,
                },
                WebLine {
                    text: "✨ আজকের নির্বাচিত তথ্য:".into(),
                    is_heading: true,
                    color: COLOR_GREEN,
                },
                WebLine {
                    text: "অপারেটিং সিস্টেম (OS) হলো একটি সিস্টেম সফটওয়্যার যা কম্পিউটারের হার্ডওয়্যার এবং সফটওয়্যার রিসোর্সগুলোর নিয়ন্ত্রণ ও সমন্বয় সাধন করে।".into(),
                    is_heading: false,
                    color: COLOR_TEXT_HIGH,
                },
                WebLine {
                    text: "অনুরণ ওএস (Onuron OS) হলো আধুনিক মাইক্রোকার্নেল ও আলাপ ফ্রেমওয়ার্ক ভিত্তিক বাংলা অপারেটিং সিস্টেম।".into(),
                    is_heading: false,
                    color: COLOR_TEXT_HIGH,
                },
            ];

            let entry = HistoryEntry {
                url: self.browser_url.clone(),
                title: self.browser_title.clone(),
                time_str: get_ist_time_str(),
            };
            self.browser_history.insert(0, entry);
            return;
        }

        if is_ddg {
            self.browser_url = "https://duckduckgo.com".into();
            self.browser_url_input = "https://duckduckgo.com".into();
            self.browser_cursor_pos = self.browser_url_input.chars().count();
            self.browser_is_editing_url = false;
            self.browser_title = "DuckDuckGo — প্রাইভেসি ও নিরাপদ সার্চ".into();
            self.browser_status = "✓ SSL TLS 1.3 | 200 OK | DuckDuckGo".into();
            self.browser_loading = false;
            self.browser_progress = 1.0;
            self.browser_scroll_offset = 0;

            self.browser_web_lines = vec![
                WebLine {
                    text: "DuckDuckGo — Privacy, Simplified.".into(),
                    is_heading: true,
                    color: COLOR_AMBER,
                },
                WebLine {
                    text: "আমরা কোনো ব্যবহারকারীর ব্যক্তিগত তথ্য বা সার্চ হিস্ট্রি ট্র্যাক করি না।".into(),
                    is_heading: false,
                    color: COLOR_TEXT_MED,
                },
                WebLine {
                    text: "🔍 যেকোনো কিছু লিখে উপরের সার্চ বারে 'গো' চাপুন।".into(),
                    is_heading: true,
                    color: COLOR_GREEN,
                },
                WebLine {
                    text: "──────── প্রাইভেসি সুবিধা ────────".into(),
                    is_heading: false,
                    color: COLOR_BORDER,
                },
                WebLine {
                    text: "• ট্র্যাকার ব্লক ও অ্যাড-ট্র্যাকিং সম্পূর্ণ প্রতিরোধ".into(),
                    is_heading: false,
                    color: COLOR_TEXT_HIGH,
                },
                WebLine {
                    text: "• সম্পূর্ণ এনক্রিপ্টেড HTTPS সংযোগ".into(),
                    is_heading: false,
                    color: COLOR_TEXT_HIGH,
                },
                WebLine {
                    text: "• জিরো-লগিং প্রাইভেট সার্চ ইঞ্জিন".into(),
                    is_heading: false,
                    color: COLOR_TEXT_HIGH,
                },
            ];

            let entry = HistoryEntry {
                url: self.browser_url.clone(),
                title: self.browser_title.clone(),
                time_str: get_ist_time_str(),
            };
            self.browser_history.insert(0, entry);
            return;
        }

        let is_direct_url = clean.contains('.') && !clean.contains(' ');
        let target_url = if is_direct_url {
            if !clean.starts_with("http://") && !clean.starts_with("https://") {
                format!("https://{}", clean)
            } else {
                clean.to_string()
            }
        } else {
            let encoded = clean.replace(' ', "+");
            format!("https://html.duckduckgo.com/html/?q={}", encoded)
        };

        self.browser_url = target_url.clone();
        self.browser_url_input = target_url.clone();
        self.browser_cursor_pos = self.browser_url_input.chars().count();
        self.browser_is_editing_url = false;
        self.browser_status = "লাইভ ইন্টারনেট থেকে লোড হচ্ছে...".into();
        self.browser_scroll_offset = 0;

        let out = Command::new("curl")
            .args(["-sL", "-m", "4", "-A", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) NilOS/1.5", &target_url])
            .output();

        self.browser_loading = false;
        self.browser_progress = 1.0;

        match out {
            Ok(res) if res.status.success() && !res.stdout.is_empty() => {
                let html = String::from_utf8_lossy(&res.stdout);
                self.browser_title = extract_html_title(&html);
                self.browser_status = "✓ SSL TLS 1.3 | 200 OK | লাইভ ওয়েব".into();

                let mut web_lines = Vec::new();
                let decoded_title = decode_html_entities(&self.browser_title);

                web_lines.push(WebLine {
                    text: decoded_title,
                    is_heading: true,
                    color: COLOR_CYAN,
                });

                let clean_text = clean_html_to_text(&html);

                for raw_line in clean_text.lines() {
                    let trimmed = raw_line.split_whitespace().collect::<Vec<_>>().join(" ");
                    if trimmed.len() > 3
                        && !trimmed.starts_with('{')
                        && !trimmed.starts_with("function")
                        && !trimmed.contains("var ")
                        && !trimmed.contains("document.")
                        && !trimmed.contains("window.")
                        && !trimmed.contains("Cookies")
                        && !trimmed.contains("CSS")
                        && !trimmed.contains("@media")
                    {
                        let is_h = trimmed.len() < 50 && (trimmed.contains("Result") || trimmed.contains("Search") || trimmed.contains("উইকিপিডিয়া") || trimmed.contains("About") || trimmed.contains("Title"));
                        web_lines.push(WebLine {
                            text: trimmed,
                            is_heading: is_h,
                            color: if is_h { COLOR_GREEN } else { COLOR_TEXT_HIGH },
                        });
                    }
                }

                if web_lines.is_empty() {
                    web_lines.push(WebLine {
                        text: format!("পৃষ্ঠা সফলভাবে লোড হয়েছে: {}", target_url),
                        is_heading: false,
                        color: COLOR_GREEN,
                    });
                }

                self.browser_web_lines = web_lines;
            }
            _ => {
                self.browser_status = "অফলাইন / সংযোগ ত্রুটি".into();
                self.browser_title = format!("{} (অফলাইন)", target_url);
                self.browser_web_lines = vec![
                    WebLine {
                        text: "ওয়েবসাইট সরাসরি লোড করা যায়নি।".into(),
                        is_heading: true,
                        color: COLOR_RED,
                    },
                    WebLine {
                        text: "অনুগ্রহ করে আপনার ইন্টারনেট সংযোগ বা URL ঠিকানাটি চেক করুন।".into(),
                        is_heading: false,
                        color: COLOR_TEXT_MED,
                    },
                ];
            }
        }

        let entry = HistoryEntry {
            url: self.browser_url.clone(),
            title: self.browser_title.clone(),
            time_str: get_ist_time_str(),
        };
        self.browser_history.insert(0, entry);
        if self.browser_history.len() > 100 {
            self.browser_history.truncate(100);
        }
    }

    fn term_char_count(&self) -> usize {
        self.term_input.chars().count()
    }

    fn term_insert_char(&mut self, ch: char) {
        let mut chars: Vec<char> = self.term_input.chars().collect();
        let cur = self.term_cursor_pos.min(chars.len());
        chars.insert(cur, ch);
        self.term_input = chars.into_iter().collect();
        self.term_cursor_pos += 1;
        self.term_scroll_offset = 0;
    }

    fn term_backspace(&mut self) {
        let mut chars: Vec<char> = self.term_input.chars().collect();
        if self.term_cursor_pos > 0 && !chars.is_empty() {
            let remove_idx = (self.term_cursor_pos - 1).min(chars.len() - 1);
            chars.remove(remove_idx);
            self.term_input = chars.into_iter().collect();
            self.term_cursor_pos = self.term_cursor_pos.saturating_sub(1);
        }
        self.term_scroll_offset = 0;
    }

    fn term_delete(&mut self) {
        let mut chars: Vec<char> = self.term_input.chars().collect();
        if self.term_cursor_pos < chars.len() {
            chars.remove(self.term_cursor_pos);
            self.term_input = chars.into_iter().collect();
        }
    }

    fn term_cursor_split(&self) -> (String, String) {
        let chars: Vec<char> = self.term_input.chars().collect();
        let cur = self.term_cursor_pos.min(chars.len());
        let before: String = chars[..cur].iter().collect();
        let after: String = chars[cur..].iter().collect();
        (before, after)
    }

    fn note_insert_char(&mut self, ch: char) {
        let mut chars: Vec<char> = self.notes_edit_content.chars().collect();
        let cur = self.notes_cursor_pos.min(chars.len());
        chars.insert(cur, ch);
        self.notes_edit_content = chars.into_iter().collect();
        self.notes_cursor_pos += 1;
    }

    fn note_backspace(&mut self) {
        let mut chars: Vec<char> = self.notes_edit_content.chars().collect();
        if self.notes_cursor_pos > 0 && !chars.is_empty() {
            let remove_idx = (self.notes_cursor_pos - 1).min(chars.len() - 1);
            chars.remove(remove_idx);
            self.notes_edit_content = chars.into_iter().collect();
            self.notes_cursor_pos = self.notes_cursor_pos.saturating_sub(1);
        }
    }

    fn browser_url_insert_char(&mut self, ch: char) {
        let mut chars: Vec<char> = self.browser_url_input.chars().collect();
        let cur = self.browser_cursor_pos.min(chars.len());
        chars.insert(cur, ch);
        self.browser_url_input = chars.into_iter().collect();
        self.browser_cursor_pos += 1;
        self.browser_is_editing_url = true;
    }

    fn browser_url_backspace(&mut self) {
        let mut chars: Vec<char> = self.browser_url_input.chars().collect();
        if self.browser_cursor_pos > 0 && !chars.is_empty() {
            let remove_idx = (self.browser_cursor_pos - 1).min(chars.len() - 1);
            chars.remove(remove_idx);
            self.browser_url_input = chars.into_iter().collect();
            self.browser_cursor_pos = self.browser_cursor_pos.saturating_sub(1);
        }
        self.browser_is_editing_url = true;
    }

    fn trigger_sms_reply(&mut self, thread_idx: usize, user_text: &str) {
        let reply = if user_text.contains("কেমন") {
            "আমি ভালো আছি! আপনি কেমন আছেন?".to_string()
        } else if user_text.contains("আসছি") || user_text.contains("আসি") {
            "খুব ভালো! আমি অপেক্ষা করছি 👍".to_string()
        } else if user_text.contains("ঠিক") {
            "ধন্যবাদ! সব ঠিকঠাক রাখা হলো ✓✓".to_string()
        } else if user_text.contains("ফাইল") {
            "ফাইলটি সফটবাসে রিসিভ হয়েছে 📁".to_string()
        } else {
            "ধন্যবাদ! আপনার বার্তা সফলভাবে পৌঁছেছে ✓✓".to_string()
        };
        self.sms_auto_reply_ticks = Some((45, thread_idx, reply));
    }

    fn push_term_line(&mut self, text: String, color: u32) {
        const MAX_COLS: usize = 38;
        let chars: Vec<char> = text.chars().collect();
        if chars.len() <= MAX_COLS {
            self.term_lines.push(TermLine { text, color });
        } else {
            let mut start = 0;
            while start < chars.len() {
                let end = (start + MAX_COLS).min(chars.len());
                let chunk: String = chars[start..end].iter().collect();
                self.term_lines.push(TermLine { text: chunk, color });
                start = end;
            }
        }
    }

    fn short_cwd(&self) -> String {
        disk_to_prompt_display(&self.storage_root, &self.term_cwd)
    }

    fn open_nano(&mut self, filename: &str) {
        self.nano_filename = filename.to_string();
        let target = virtual_to_disk_path(&self.storage_root, &self.term_cwd, filename);
        if target.exists() {
            if let Ok(content) = fs::read_to_string(&target) {
                self.nano_lines = content.lines().map(|s| s.to_string()).collect();
                if self.nano_lines.is_empty() {
                    self.nano_lines.push("".to_string());
                }
            } else {
                self.nano_lines = vec!["".to_string()];
            }
        } else {
            self.nano_lines = vec!["".to_string()];
        }
        self.nano_row = 0;
        self.nano_col = 0;
        self.nano_dirty = false;
        self.nano_status = format!("GNU nano 7.2 | ফাইল: {}", filename);
        self.screen = Screen::NanoEditor;
    }

    fn nano_save(&mut self) {
        let target = virtual_to_disk_path(&self.storage_root, &self.term_cwd, &self.nano_filename);
        let content = self.nano_lines.join("\n");
        match fs::write(&target, content) {
            Ok(_) => {
                self.nano_dirty = false;
                self.nano_status = format!("[সংরক্ষিত: {} টি লাইন]", self.nano_lines.len());
            }
            Err(e) => {
                self.nano_status = format!("[সংরক্ষণ ব্যর্থ: {}]", e);
            }
        }
    }

    fn nano_insert_char(&mut self, ch: char) {
        if self.nano_row >= self.nano_lines.len() {
            self.nano_lines.push("".to_string());
        }
        let line = &self.nano_lines[self.nano_row];
        let mut chars: Vec<char> = line.chars().collect();
        let col = self.nano_col.min(chars.len());
        chars.insert(col, ch);
        self.nano_lines[self.nano_row] = chars.into_iter().collect();
        self.nano_col += 1;
        self.nano_dirty = true;
    }

    fn nano_enter(&mut self) {
        if self.nano_row >= self.nano_lines.len() {
            self.nano_lines.push("".to_string());
            self.nano_row += 1;
            self.nano_col = 0;
            return;
        }
        let line = &self.nano_lines[self.nano_row];
        let chars: Vec<char> = line.chars().collect();
        let col = self.nano_col.min(chars.len());
        let before: String = chars[..col].iter().collect();
        let after: String = chars[col..].iter().collect();
        self.nano_lines[self.nano_row] = before;
        self.nano_lines.insert(self.nano_row + 1, after);
        self.nano_row += 1;
        self.nano_col = 0;
        self.nano_dirty = true;
    }

    fn nano_backspace(&mut self) {
        if self.nano_row >= self.nano_lines.len() {
            return;
        }
        if self.nano_col > 0 {
            let line = &self.nano_lines[self.nano_row];
            let mut chars: Vec<char> = line.chars().collect();
            let remove_idx = self.nano_col - 1;
            if remove_idx < chars.len() {
                chars.remove(remove_idx);
                self.nano_lines[self.nano_row] = chars.into_iter().collect();
                self.nano_col -= 1;
                self.nano_dirty = true;
            }
        } else if self.nano_row > 0 {
            let cur_line = self.nano_lines.remove(self.nano_row);
            self.nano_row -= 1;
            let prev_len = self.nano_lines[self.nano_row].chars().count();
            self.nano_lines[self.nano_row].push_str(&cur_line);
            self.nano_col = prev_len;
            self.nano_dirty = true;
        }
    }

    fn exec_calc_press(&mut self, btn: &str) {
        if btn == "C" {
            self.calc_expr = "০".into();
            self.calc_result.clear();
        } else if btn == "=" {
            let ascii_expr = self.calc_expr
                .replace('০', "0").replace('১', "1").replace('২', "2").replace('৩', "3")
                .replace('৪', "4").replace('৫', "5").replace('৬', "6").replace('৭', "7")
                .replace('৮', "8").replace('৯', "9").replace('×', "*").replace('÷', "/");
            
            if let Ok(val) = eval_simple_math(&ascii_expr) {
                let res_str = if val.fract() == 0.0 {
                    format!("{:.0}", val)
                } else {
                    format!("{:.2}", val)
                };
                let formatted = to_bengali_digits(&res_str);
                self.calc_result = formatted.clone();
                self.calc_history.push((self.calc_expr.clone(), formatted));
                if self.calc_history.len() > 8 {
                    self.calc_history.remove(0);
                }
            } else {
                self.calc_result = "ত্রুটি".into();
            }
        } else if btn == "±" {
            if self.calc_expr.starts_with('-') {
                self.calc_expr.remove(0);
            } else if self.calc_expr != "০" {
                self.calc_expr.insert(0, '-');
            }
        } else if btn == "( )" {
            let open_count = self.calc_expr.chars().filter(|&c| c == '(').count();
            let close_count = self.calc_expr.chars().filter(|&c| c == ')').count();
            if open_count > close_count && !self.calc_expr.ends_with('(') && !self.calc_expr.ends_with('+') && !self.calc_expr.ends_with('-') && !self.calc_expr.ends_with('×') && !self.calc_expr.ends_with('÷') {
                self.calc_expr.push(')');
            } else if self.calc_expr == "০" {
                self.calc_expr = "(".into();
            } else {
                self.calc_expr.push('(');
            }
        } else {
            if self.calc_expr == "০" && btn != "." && btn != "+" && btn != "-" && btn != "×" && btn != "÷" && btn != "%" {
                self.calc_expr = btn.to_string();
            } else {
                self.calc_expr.push_str(btn);
            }
        }
    }



    fn exec_term_command(&mut self, raw_cmd: &str) {
        let cmd = raw_cmd.trim();
        if cmd.is_empty() {
            return;
        }

        if self.python_mode {
            let prompt = format!("py>>> {}", cmd);
            self.push_term_line(prompt, COLOR_AMBER);
            self.term_history.push(cmd.to_string());
            self.term_history_idx = None;
            self.term_scroll_offset = 0;

            if cmd == "exit()" || cmd == "quit()" || cmd == "exit" || cmd == "quit" {
                self.python_mode = false;
                self.push_term_line("[পাইথন REPL সমাপ্ত]".into(), COLOR_CYAN);
                return;
            }

            let py_code = format!(
                "import sys\ntry:\n    _res = eval({:?})\n    if _res is not None: print(_res)\nexcept Exception:\n    try:\n        exec({:?})\n    except Exception as e:\n        print(f'ত্রুটি: {{e}}', file=sys.stderr)",
                cmd, cmd
            );

            match Command::new("python")
                .arg("-c")
                .arg(&py_code)
                .current_dir(&self.term_cwd)
                .output()
            {
                Ok(out) => {
                    let stdout = String::from_utf8_lossy(&out.stdout);
                    for line in stdout.lines() {
                        self.push_term_line(line.to_string(), COLOR_GREEN);
                    }
                    let stderr = String::from_utf8_lossy(&out.stderr);
                    for line in stderr.lines() {
                        self.push_term_line(line.to_string(), COLOR_RED);
                    }
                }
                Err(e) => {
                    self.push_term_line(format!("পাইথন রান করতে ব্যর্থ: {}", e), COLOR_RED);
                }
            }
            return;
        }

        let prompt = format!("joy@nilos:{}$ {}", self.short_cwd(), cmd);
        self.push_term_line(prompt, COLOR_CYAN);

        self.term_history.push(cmd.to_string());
        self.term_history_idx = None;
        self.term_scroll_offset = 0;

        // Builtin: NilVM / NilLang Runner & Compiler Tools
        if cmd == "nil" || cmd.starts_with("nil ") || cmd == "nilvm" || cmd.starts_with("nilvm ") {
            let args: Vec<&str> = cmd.split_whitespace().collect();
            if args.len() == 1 || (args.len() == 2 && (args[1] == "help" || args[1] == "-h" || args[1] == "--help")) {
                self.push_term_line("NilVM & NilLang Bytecode Engine v2.0 (Onuron OS)".into(), COLOR_CYAN);
                self.push_term_line("ব্যবহার:".into(), COLOR_AMBER);
                self.push_term_line("  nil run <file.nil>     — NilLang স্ক্রিপ্ট রান করুন".into(), COLOR_TEXT_HIGH);
                self.push_term_line("  nil hir <file.nil>     — High-level IR ট্রি পরিদর্শন".into(), COLOR_TEXT_HIGH);
                self.push_term_line("  nil disasm <file.nil>  — NABC বাইটকোড ডিসঅ্যাসেম্বল".into(), COLOR_TEXT_HIGH);
                self.push_term_line("  nilpos print <item>    — POS থার্মাল রসিদ প্রিন্ট".into(), COLOR_TEXT_HIGH);
                self.push_term_line("  softbus list           — SoftBus পিয়ার নোড তালিকা".into(), COLOR_TEXT_HIGH);
                self.push_term_line("  nilhal info            — ভার্চুয়ালাইজড হার্ডওয়্যার তথ্য".into(), COLOR_TEXT_HIGH);
                return;
            }

            let subcmd = args[1];
            if subcmd == "hir" {
                if let Some(&filename) = args.get(2) {
                    let target = virtual_to_disk_path(&self.storage_root, &self.term_cwd, filename);
                    match fs::read_to_string(&target) {
                        Ok(content) => {
                            let hir_lines = nilvm::generate_hir_tree(filename, &content);
                            for hl in hir_lines {
                                self.push_term_line(hl, COLOR_AMBER);
                            }
                        }
                        Err(_) => {
                            self.push_term_line(format!("ত্রুটি: ফাইল পাওয়া যায়নি '{}'", filename), COLOR_RED);
                        }
                    }
                } else {
                    self.push_term_line("ব্যবহার: nil hir <file.nil>".into(), COLOR_RED);
                }
                return;
            }

            if subcmd == "disasm" || subcmd == "asm" {
                if let Some(&filename) = args.get(2) {
                    let target = virtual_to_disk_path(&self.storage_root, &self.term_cwd, filename);
                    match fs::read_to_string(&target) {
                        Ok(content) => {
                            let disasm_lines = nilvm::disassemble_nabc(filename, &content);
                            for dl in disasm_lines {
                                self.push_term_line(dl, COLOR_CYAN);
                            }
                        }
                        Err(_) => {
                            self.push_term_line(format!("ত্রুটি: ফাইল পাওয়া যায়নি '{}'", filename), COLOR_RED);
                        }
                    }
                } else {
                    self.push_term_line("ব্যবহার: nil disasm <file.nil>".into(), COLOR_RED);
                }
                return;
            }

            if subcmd == "run" || args.len() == 2 {
                let filename = if subcmd == "run" {
                    args.get(2).copied().unwrap_or("")
                } else {
                    subcmd
                };

                if filename.is_empty() {
                    self.push_term_line("ব্যবহার: nil run <file.nil>".into(), COLOR_RED);
                    return;
                }

                let target = virtual_to_disk_path(&self.storage_root, &self.term_cwd, filename);
                match fs::read_to_string(&target) {
                    Ok(content) => {
                        self.push_term_line(format!("[NilVM 2.0] এক্সিকিউটিং: {}...", filename), COLOR_CYAN);
                        let start_time = std::time::Instant::now();
                        let mut vm = NilVM::new();
                        match vm.execute_source(&content) {
                            Ok(()) => {
                                for out in vm.output_lines {
                                    self.push_term_line(out, COLOR_GREEN);
                                }
                                let elapsed = start_time.elapsed().as_secs_f64() * 1000.0;
                                self.push_term_line(format!("[✓ এক্সিকিউশন সফল: {:.2} ms | গ্লোবালস: {} | স্ট্যাক: ০]", elapsed, vm.globals.len()), COLOR_TEAL);
                            }
                            Err(e) => {
                                self.push_term_line(format!("[!] NilVM রানটাইম ত্রুটি: {}", e), COLOR_RED);
                            }
                        }
                    }
                    Err(_) => {
                        self.push_term_line(format!("ত্রুটি: ফাইল পাওয়া যায়নি '{}'", filename), COLOR_RED);
                    }
                }
                return;
            }
        }

        // Builtin: nilpos Thermal Receipt Printer
        if cmd == "nilpos" || cmd.starts_with("nilpos ") {
            self.push_term_line("========================================".into(), COLOR_CYAN);
            self.push_term_line("        অনুরণ পিওএস ক্যাশ রসিদ         ".into(), COLOR_AMBER);
            self.push_term_line("        রসিদ নং: #NPOS-9824            ".into(), COLOR_TEXT_MED);
            self.push_term_line("----------------------------------------".into(), COLOR_BORDER);
            self.push_term_line("পণ্য / বিবরণ                    মূল্য    ".into(), COLOR_TEXT_HIGH);
            self.push_term_line("----------------------------------------".into(), COLOR_BORDER);
            self.push_term_line("১. NilPhone Ultra ওএস কিট       ₹ ২৪,৯৯৯ ".into(), COLOR_GREEN);
            self.push_term_line("২. SoftBus ডঙ্গল এডাপ্টার        ₹ ১,২০০   ".into(), COLOR_GREEN);
            self.push_term_line("----------------------------------------".into(), COLOR_BORDER);
            self.push_term_line("মোট মূল্য:                     ₹ ২৬,১৯৯ ".into(), COLOR_TEXT_HIGH);
            self.push_term_line("জিএসটি (১৮%):                   ₹ ৪,৭১৫.৮২".into(), COLOR_TEXT_MED);
            self.push_term_line("সর্বমোট প্রদেয়:                ₹ ৩০,৯১৪.৮২".into(), COLOR_AMBER);
            self.push_term_line("স্ট্যাটাস: পরিশোধিত [SoftBus Pay / UPI]  ".into(), COLOR_TEAL);
            self.push_term_line("========================================".into(), COLOR_CYAN);
            return;
        }

        // Builtin: SoftBus Mesh Network Status
        if cmd == "softbus" || cmd.starts_with("softbus ") {
            self.push_term_line("==================================================".into(), COLOR_CYAN);
            self.push_term_line("  SoftBus Distributed Peer Mesh (Onuron OS v2.0)  ".into(), COLOR_AMBER);
            self.push_term_line("==================================================".into(), COLOR_CYAN);
            self.push_term_line("  [✓] Node #1: Desktop-Host (Windows 11 / x86_64)   - 0.4 ms".into(), COLOR_GREEN);
            self.push_term_line("  [✓] Node #2: NilPhone-Ultra (Onuron ARM64)        - 1.2 ms".into(), COLOR_GREEN);
            self.push_term_line("  [✓] Node #3: NilOS-Tab Pro (Tablet Display Node)  - 2.8 ms".into(), COLOR_GREEN);
            self.push_term_line("  [✓] Bus Security: AES-256-GCM Zero-Trust Channel".into(), COLOR_TEAL);
            self.push_term_line("  [✓] Synchronized Services: Audio, Clipboard, VFS".into(), COLOR_TEXT_MED);
            self.push_term_line("==================================================".into(), COLOR_BORDER);
            return;
        }

        // Builtin: NilHAL Hardware Virtualization Info
        if cmd == "nilhal" || cmd.starts_with("nilhal ") {
            self.push_term_line("==================================================".into(), COLOR_CYAN);
            self.push_term_line("  NilHAL Hardware Virtualization Subsystem v2.0   ".into(), COLOR_AMBER);
            self.push_term_line("==================================================".into(), COLOR_CYAN);
            self.push_term_line("  • GPU/Display: NilScreen 330x680 @ 60Hz (Subpixel AA)".into(), COLOR_GREEN);
            self.push_term_line("  • Audio Engine: 4-Voice Polyphonic Synthesizer + DSP".into(), COLOR_GREEN);
            self.push_term_line("  • Storage:      Flash VFS (/home/joy, 64 GB Virtual)".into(), COLOR_GREEN);
            self.push_term_line("  • Network NIC:  SoftBus Virtual Fabric (10 Gbps VirtIO)".into(), COLOR_GREEN);
            self.push_term_line("  • Input Bridge: Multi-Touch Digitizer & Keyboard".into(), COLOR_GREEN);
            self.push_term_line("==================================================".into(), COLOR_BORDER);
            return;
        }

        // Builtin: VLC Media Player CLI
        if cmd == "vlc" || cmd.starts_with("vlc ") {
            self.install_pkg("org.videolan.vlc");
            if cmd.starts_with("vlc ") {
                let _target = cmd[4..].trim();
                self.push_term_line("[*] মিডিয়া চালাতে ব্রাউজারে খুলুন।".into(), COLOR_TEXT_MED);
            }
            self.screen = Screen::AppBrowser;
            return;
        }

        // Builtin: nilpkg package manager CLI
        if cmd == "nilpkg" || cmd.starts_with("nilpkg ") {
            let args: Vec<&str> = cmd.split_whitespace().collect();
            if args.len() == 1 || args.get(1) == Some(&"help") {
                self.push_term_line("NilPkg Package Manager v1.5 (Official Onuron OS)".into(), COLOR_CYAN);
                self.push_term_line("ব্যবহার: nilpkg install <pkg>, nilpkg list, nilpkg search".into(), COLOR_TEXT_MED);
                return;
            }
            if args.get(1) == Some(&"list") {
                self.push_term_line("ইনস্টলকৃত প্যাকেজসমূহ:".into(), COLOR_CYAN);
                let pkgs = self.installed_pkgs.clone();
                for p in pkgs {
                    self.push_term_line(format!("  • {}", p), COLOR_GREEN);
                }
                return;
            }
            if args.get(1) == Some(&"install") {
                if let Some(&pkg_name) = args.get(2) {
                    let target_id = match pkg_name {
                        "vlc" | "videolan" | "org.videolan.vlc" => "org.videolan.vlc",
                        "firefox" | "fenix" | "org.mozilla.fenix" => "org.mozilla.fenix",
                        "signal" | "com.signal.android" => "com.signal.android",
                        other => other,
                    };
                    self.push_term_line(format!("[*] ডাউনলোড ও ইনস্টল করা হচ্ছে: {}...", target_id), COLOR_AMBER);
                    self.install_pkg(target_id);
                    return;
                }
            }
        }

        // Builtin: firefox / browser command
        if cmd == "firefox" || cmd.starts_with("firefox ") || cmd == "browser" || cmd.starts_with("browser ") {
            self.install_pkg("org.mozilla.fenix");
            let url = if cmd.starts_with("firefox ") {
                cmd[8..].trim()
            } else if cmd.starts_with("browser ") {
                cmd[8..].trim()
            } else {
                "https://google.com"
            };
            self.browser_fetch_live(url);
            self.screen = Screen::AppBrowser;
            return;
        }

        if cmd == "nano" || cmd.starts_with("nano ") || cmd.starts_with("edit ") {
            let filename = if cmd == "nano" {
                "untitled.txt"
            } else if cmd.starts_with("nano ") {
                cmd[5..].trim()
            } else {
                cmd[5..].trim()
            };
            self.open_nano(filename);
            return;
        }

        if cmd == "python" || cmd == "py" || cmd == "python3" {
            self.python_mode = true;
            self.push_term_line("Python 3.14 Interactive REPL (NilOS)".into(), COLOR_AMBER);
            self.push_term_line("যেকোনো কোড টাইপ করুন, বের হতে 'exit()' লিখুন।".into(), COLOR_TEXT_MED);
            return;
        }

        if cmd == "clear" || cmd == "cls" {
            self.term_lines.clear();
            return;
        }

        if cmd == "pwd" {
            let cwd_str = disk_to_virtual_display(&self.storage_root, &self.term_cwd);
            self.push_term_line(cwd_str, COLOR_GREEN);
            return;
        }

        if cmd.starts_with("mkdir ") || cmd.starts_with("md ") {
            let dir_name = if cmd.starts_with("mkdir ") {
                cmd[6..].trim()
            } else {
                cmd[3..].trim()
            };
            if !dir_name.is_empty() {
                let target = virtual_to_disk_path(&self.storage_root, &self.term_cwd, dir_name);
                match fs::create_dir_all(&target) {
                    Ok(_) => {
                        self.push_term_line(format!("[+] নতুন ফোল্ডার তৈরি হয়েছে: {}", dir_name), COLOR_GREEN);
                    }
                    Err(e) => {
                        self.push_term_line(format!("mkdir ত্রুটি: {}", e), COLOR_RED);
                    }
                }
                return;
            }
        }

        if cmd == "ls" || cmd.starts_with("ls ") || cmd == "dir" || cmd.starts_with("dir ") {
            let mut show_all = false;
            let mut path_arg = "";
            let args: Vec<&str> = cmd.split_whitespace().skip(1).collect();
            for arg in &args {
                if *arg == "-a" || *arg == "-la" || *arg == "-al" {
                    show_all = true;
                } else if !arg.starts_with('-') {
                    path_arg = *arg;
                }
            }

            let target_dir = if path_arg.is_empty() {
                self.term_cwd.clone()
            } else {
                virtual_to_disk_path(&self.storage_root, &self.term_cwd, path_arg)
            };

            match fs::read_dir(&target_dir) {
                Ok(entries) => {
                    let mut dirs = Vec::new();
                    let mut files = Vec::new();

                    for entry in entries.flatten() {
                        let name = entry.file_name().to_string_lossy().to_string();
                        if !show_all && name.starts_with('.') {
                            continue;
                        }
                        if let Ok(ft) = entry.file_type() {
                            if ft.is_dir() {
                                dirs.push(name);
                            } else {
                                let size_kb = entry.metadata().map(|m| m.len() as f64 / 1024.0).unwrap_or(0.0);
                                files.push((name, size_kb));
                            }
                        }
                    }

                    dirs.sort();
                    files.sort_by(|a, b| a.0.cmp(&b.0));

                    let ls_detailed = cmd.contains("-l") || cmd.contains("-la") || cmd.contains("-al");
                    let total = dirs.len() + files.len();

                    if dirs.is_empty() && files.is_empty() {
                        self.push_term_line("  (ফোল্ডার ফাঁকা)".into(), COLOR_TEXT_DIM);
                    } else {
                        if ls_detailed {
                            self.push_term_line(format!("total {}", total * 4), COLOR_TEXT_DIM);
                        }
                        for d in &dirs {
                            if ls_detailed {
                                self.push_term_line(
                                    format!("drwxr-xr-x  2 joy  nil   4096 Sep  1 15:40 {}/", d),
                                    COLOR_CYAN
                                );
                            } else {
                                self.push_term_line(format!("  {}/", d), COLOR_CYAN);
                            }
                        }
                        for (f, sz) in &files {
                            let size_str = if *sz < 1.0 {
                                format!("{:>8.0}B", sz * 1024.0)
                            } else if *sz < 1024.0 {
                                format!("{:>7.1}K", sz)
                            } else {
                                format!("{:>7.1}M", sz / 1024.0)
                            };
                            let (color, icon) = if f.ends_with(".sh") || f.ends_with(".py") {
                                (COLOR_GREEN, "-rwxr-xr-x")
                            } else {
                                (COLOR_TEXT_HIGH, "-rw-r--r--")
                            };
                            if ls_detailed {
                                self.push_term_line(
                                    format!("{}  1 joy  nil {} Sep  1 15:40 {}", icon, size_str, f),
                                    color
                                );
                            } else {
                                self.push_term_line(format!("  {}", f), color);
                            }
                        }
                    }
                }
                Err(e) => {
                    self.push_term_line(format!("ls ত্রুটি: {}", e), COLOR_RED);
                }
            }
            return;
        }

        if cmd.starts_with("cat ") || cmd.starts_with("type ") {
            let filename = if cmd.starts_with("cat ") {
                cmd[4..].trim()
            } else {
                cmd[5..].trim()
            };
            let target = virtual_to_disk_path(&self.storage_root, &self.term_cwd, filename);
            match fs::read_to_string(&target) {
                Ok(content) => {
                    for line in content.lines().take(60) {
                        self.push_term_line(line.to_string(), COLOR_TEXT_HIGH);
                    }
                }
                Err(_) => {
                    self.push_term_line(format!("cat: {}: No such file or directory", filename), COLOR_RED);
                }
            }
            return;
        }

        if cmd.starts_with("touch ") {
            let filename = cmd[6..].trim();
            let target = virtual_to_disk_path(&self.storage_root, &self.term_cwd, filename);
            match fs::File::create(&target) {
                Ok(_) => {
                    self.push_term_line(format!("[+] নতুন ফাইল তৈরি হয়েছে: {}", filename), COLOR_GREEN);
                }
                Err(e) => {
                    self.push_term_line(format!("touch ত্রুটি: {}", e), COLOR_RED);
                }
            }
            return;
        }

        if cmd.starts_with("rm ") || cmd.starts_with("del ") {
            let filename = if cmd.starts_with("rm ") {
                cmd[3..].trim()
            } else {
                cmd[4..].trim()
            };
            let target = virtual_to_disk_path(&self.storage_root, &self.term_cwd, filename);
            if target.is_dir() {
                match fs::remove_dir_all(&target) {
                    Ok(_) => self.push_term_line(format!("[-] ফোল্ডার মোছা হয়েছে: {}", filename), COLOR_AMBER),
                    Err(e) => self.push_term_line(format!("rm ত্রুটি: {}", e), COLOR_RED),
                }
            } else {
                match fs::remove_file(&target) {
                    Ok(_) => self.push_term_line(format!("[-] ফাইল মোছা হয়েছে: {}", filename), COLOR_AMBER),
                    Err(e) => self.push_term_line(format!("rm ত্রুটি: {}", e), COLOR_RED),
                }
            }
            return;
        }

        if cmd.starts_with("cd ") || cmd == "cd" || cmd == "cd.." || cmd == "cd." {
            let target_str = if cmd == "cd" {
                "~".to_string()
            } else if cmd == "cd.." {
                "..".to_string()
            } else if cmd == "cd." {
                ".".to_string()
            } else {
                cmd[3..].trim().to_string()
            };

            let new_path = virtual_to_disk_path(&self.storage_root, &self.term_cwd, &target_str);
            if new_path.is_dir() {
                self.term_cwd = new_path;
                self.current_path = self.term_cwd.to_string_lossy().to_string();
            } else {
                self.push_term_line(format!("cd: '{}': No such file or directory", target_str), COLOR_RED);
            }
            return;
        }

        let result = if let Some(ref bash) = self.bash_path {
            Command::new(bash)
                .arg("-c")
                .arg(cmd)
                .current_dir(&self.term_cwd)
                .output()
        } else {
            #[cfg(target_os = "windows")]
            let res = Command::new("powershell")
                .arg("-NoProfile")
                .arg("-Command")
                .arg(cmd)
                .current_dir(&self.term_cwd)
                .output();

            #[cfg(not(target_os = "windows"))]
            let res = Command::new("sh")
                .arg("-c")
                .arg(cmd)
                .current_dir(&self.term_cwd)
                .output();

            res
        };

        match result {
            Ok(output) => {
                let stdout = String::from_utf8_lossy(&output.stdout);
                for line in stdout.lines() {
                    self.push_term_line(line.to_string(), COLOR_GREEN);
                }
                let stderr = String::from_utf8_lossy(&output.stderr);
                for line in stderr.lines() {
                    self.push_term_line(line.to_string(), COLOR_RED);
                }
                if !output.status.success() {
                    if let Some(code) = output.status.code() {
                        if code != 0 {
                            self.push_term_line(format!("[প্রসেস সমাপ্তি কোড: {}]", code), COLOR_AMBER);
                        }
                    }
                }
            }
            Err(e) => {
                self.push_term_line(format!("কমান্ড চালানো যায়নি: {}", e), COLOR_RED);
            }
        }

        while self.term_lines.len() > 200 {
            self.term_lines.remove(0);
        }
    }

    fn new_browser_tab(&mut self, url: &str) {
        let id = self.browser_next_tab_id;
        self.browser_next_tab_id += 1;
        let title = if url.contains("google") {
            "Google".to_string()
        } else if url.contains("youtube") {
            "YouTube".to_string()
        } else if url.contains("wikipedia") {
            "Wikipedia".to_string()
        } else {
            url.trim_start_matches("https://").trim_start_matches("http://").chars().take(12).collect::<String>()
        };
        self.browser_tabs.push(BrowserTab {
            id,
            title,
            url: url.to_string(),
        });
        self.browser_active_tab = self.browser_tabs.len() - 1;
        self.browser_url = url.to_string();
        self.browser_url_input = url.to_string();
        self.browser_is_editing_url = false;
    }

    fn close_browser_tab(&mut self, idx: usize) {
        if self.browser_tabs.len() <= 1 {
            self.browser_tabs[0] = BrowserTab {
                id: 1,
                title: "Google".into(),
                url: "https://www.google.com".into(),
            };
            self.browser_active_tab = 0;
            self.browser_url = "https://www.google.com".into();
            self.browser_url_input = "https://www.google.com".into();
            return;
        }
        self.browser_tabs.remove(idx);
        if self.browser_active_tab >= self.browser_tabs.len() {
            self.browser_active_tab = self.browser_tabs.len() - 1;
        }
        if let Some(tab) = self.browser_tabs.get(self.browser_active_tab) {
            self.browser_url = tab.url.clone();
            self.browser_url_input = tab.url.clone();
        }
    }

    fn switch_browser_tab(&mut self, idx: usize) {
        if idx < self.browser_tabs.len() {
            self.browser_active_tab = idx;
            self.browser_url = self.browser_tabs[idx].url.clone();
            self.browser_url_input = self.browser_tabs[idx].url.clone();
            self.browser_is_editing_url = false;
        }
    }
}

fn eval_simple_math(expr: &str) -> Result<f64, ()> {
    let mut clean = expr.trim().replace(' ', "");
    if clean.is_empty() {
        return Err(());
    }

    // Strip matched outer parentheses
    while clean.starts_with('(') && clean.ends_with(')') {
        let mut depth = 0;
        let mut valid_outer = true;
        for (i, c) in clean.chars().enumerate() {
            if c == '(' { depth += 1; }
            else if c == ')' {
                depth -= 1;
                if depth == 0 && i < clean.len() - 1 {
                    valid_outer = false;
                    break;
                }
            }
        }
        if valid_outer && depth == 0 {
            clean = clean[1..clean.len() - 1].trim().to_string();
        } else {
            break;
        }
    }

    // Direct number parse
    if let Some(inner) = clean.strip_suffix('%') {
        if let Ok(v) = inner.parse::<f64>() {
            return Ok(v / 100.0);
        }
    }
    if let Ok(v) = clean.parse::<f64>() {
        return Ok(v);
    }

    // Find lowest-precedence operator outside parentheses (+ and - first from right, then * and /)
    let chars: Vec<char> = clean.chars().collect();
    for ops in [&['+', '-'][..], &['*', '/'][..]] {
        let mut depth = 0;
        for i in (1..chars.len().saturating_sub(1)).rev() {
            let c = chars[i];
            if c == ')' {
                depth += 1;
            } else if c == '(' {
                depth -= 1;
            } else if depth == 0 && ops.contains(&c) {
                let left_str: String = chars[..i].iter().collect();
                let right_str: String = chars[i + 1..].iter().collect();
                let left = eval_simple_math(&left_str)?;
                let right = eval_simple_math(&right_str)?;
                let res = match c {
                    '+' => left + right,
                    '-' => left - right,
                    '*' => left * right,
                    '/' => if right != 0.0 { left / right } else { return Err(()); },
                    _ => return Err(()),
                };
                return Ok(res);
            }
        }
    }

    Err(())
}


fn draw_home_vector_icon(p: &mut FramePainter, cx: i16, cy: i16, id: &str, accent: u32) {
    draw_vector_icon(p, cx, cy, id, accent, 24);
}

fn draw_vector_icon(p: &mut FramePainter, cx: i16, cy: i16, id: &str, accent: u32, _size: u16) {
    match id {
        "media_play" | "play" => {
            // Crisp anti-aliased right-pointing triangle ▶
            p.fill_triangle_aa(
                (cx as f32 - 4.0, cy as f32 - 7.5),
                (cx as f32 - 4.0, cy as f32 + 7.5),
                (cx as f32 + 7.5, cy as f32),
                accent,
            );
        }
        "media_pause" | "pause" => {
            // Two crisp vertical pill bars ❚❚
            p.fill_rounded_rect(cx - 6, cy - 7, 4, 14, 2, accent);
            p.fill_rounded_rect(cx + 2, cy - 7, 4, 14, 2, accent);
        }
        "media_prev" | "prev" => {
            // Left bar + two anti-aliased left-pointing triangles |◀◀
            p.fill_rounded_rect(cx - 8, cy - 6, 2, 12, 1, accent);
            p.fill_triangle_aa(
                (cx as f32, cy as f32 - 6.0),
                (cx as f32, cy as f32 + 6.0),
                (cx as f32 - 6.0, cy as f32),
                accent,
            );
            p.fill_triangle_aa(
                (cx as f32 + 6.0, cy as f32 - 6.0),
                (cx as f32 + 6.0, cy as f32 + 6.0),
                (cx as f32, cy as f32),
                accent,
            );
        }
        "media_next" | "next" => {
            // Two anti-aliased right-pointing triangles + right bar ▶▶|
            p.fill_triangle_aa(
                (cx as f32 - 6.0, cy as f32 - 6.0),
                (cx as f32 - 6.0, cy as f32 + 6.0),
                (cx as f32, cy as f32),
                accent,
            );
            p.fill_triangle_aa(
                (cx as f32, cy as f32 - 6.0),
                (cx as f32, cy as f32 + 6.0),
                (cx as f32 + 6.0, cy as f32),
                accent,
            );
            p.fill_rounded_rect(cx + 6, cy - 6, 2, 12, 1, accent);
        }
        "media_vol_down" | "vol_down" => {
            // Anti-aliased speaker body + 1 wave arc
            p.fill_rounded_rect(cx - 7, cy - 3, 3, 6, 1, accent);
            p.fill_triangle_aa(
                (cx as f32 - 4.0, cy as f32 - 3.0),
                (cx as f32 - 4.0, cy as f32 + 3.0),
                (cx as f32, cy as f32 - 6.0),
                accent,
            );
            p.fill_triangle_aa(
                (cx as f32 - 4.0, cy as f32 + 3.0),
                (cx as f32, cy as f32 + 6.0),
                (cx as f32, cy as f32 - 6.0),
                accent,
            );
            p.draw_line_aa(cx as f32 + 3.0, cy as f32 - 3.5, cx as f32 + 3.0, cy as f32 + 3.5, 2.0, accent);
        }
        "media_vol_up" | "vol_up" => {
            // Anti-aliased speaker body + 2 wave arcs
            p.fill_rounded_rect(cx - 8, cy - 3, 3, 6, 1, accent);
            p.fill_triangle_aa(
                (cx as f32 - 5.0, cy as f32 - 3.0),
                (cx as f32 - 5.0, cy as f32 + 3.0),
                (cx as f32 - 1.0, cy as f32 - 6.0),
                accent,
            );
            p.fill_triangle_aa(
                (cx as f32 - 5.0, cy as f32 + 3.0),
                (cx as f32 - 1.0, cy as f32 + 6.0),
                (cx as f32 - 1.0, cy as f32 - 6.0),
                accent,
            );
            p.draw_line_aa(cx as f32 + 2.5, cy as f32 - 4.0, cx as f32 + 2.5, cy as f32 + 4.0, 1.8, accent);
            p.draw_line_aa(cx as f32 + 6.5, cy as f32 - 6.5, cx as f32 + 6.5, cy as f32 + 6.5, 1.8, accent);
        }
        "app_phone" | "phone" => {
            // High-def Smartphone Handset: curved receiver & earpieces
            p.fill_rounded_rect(cx - 9, cy - 11, 8, 7, 3, accent);
            p.fill_rounded_rect(cx + 1, cy + 4, 8, 7, 3, accent);
            p.draw_line_aa(cx as f32 - 5.0, cy as f32 - 7.0, cx as f32 + 5.0, cy as f32 + 7.0, 6.0, accent);
            p.fill_circle_aa(cx - 5, cy - 8, 1.8, 0xFFFFFF);
            p.fill_circle_aa(cx + 5, cy + 7, 1.8, 0xFFFFFF);
        }
        "app_messages" | "messages" => {
            // Modern Material 3 Message Bubble with anti-aliased speech tail
            p.fill_rounded_rect(cx - 12, cy - 10, 24, 17, 7, accent);
            p.fill_triangle_aa(
                (cx as f32 - 9.0, cy as f32 + 5.0),
                (cx as f32 - 3.0, cy as f32 + 5.0),
                (cx as f32 - 11.0, cy as f32 + 10.0),
                accent,
            );
            // 3 Modern dots
            p.fill_circle_aa(cx - 6, cy - 1, 1.8, M3_SURFACE);
            p.fill_circle_aa(cx, cy - 1, 1.8, M3_SURFACE);
            p.fill_circle_aa(cx + 6, cy - 1, 1.8, M3_SURFACE);
        }
        "app_browser" | "browser" => {
            // Glowing Cyber Globe
            p.fill_circle_aa(cx, cy, 11.0, accent);
            p.fill_circle_aa(cx, cy, 9.0, M3_SURFACE);
            p.draw_line_aa(cx as f32 - 9.0, cy as f32, cx as f32 + 9.0, cy as f32, 1.5, accent);
            p.draw_circle_outline_aa(cx, cy, 5.5, 1.5, accent);
            p.fill_circle_aa(cx, cy, 1.5, 0xFFFFFF);
        }
        "app_files" | "files" => {
            // Modern Layered Folder & Document
            p.fill_rounded_rect(cx - 11, cy - 10, 10, 5, 2, accent);
            p.fill_rounded_rect(cx - 12, cy - 7, 24, 18, 4, accent);
            p.fill_rounded_rect(cx - 7, cy - 6, 14, 8, 2, 0xFFFFFF);
            p.fill_rounded_rect(cx - 12, cy - 2, 24, 13, 3, M3_BLUE);
            p.fill_rect(cx - 5, cy - 4, 10, 1, 0x64748B);
        }
        "app_notes" | "notes" => {
            // Warm Amber Notepad with pencil
            p.fill_rounded_rect(cx - 10, cy - 11, 20, 22, 4, accent);
            p.fill_rounded_rect(cx - 8, cy - 9, 16, 18, 2, 0xFEF3C7);
            p.draw_line_aa(cx as f32 - 5.0, cy as f32 - 5.0, cx as f32 + 5.0, cy as f32 - 5.0, 1.8, 0xD97706);
            p.draw_line_aa(cx as f32 - 5.0, cy as f32 - 1.0, cx as f32 + 5.0, cy as f32 - 1.0, 1.8, 0xD97706);
            p.draw_line_aa(cx as f32 - 5.0, cy as f32 + 3.0, cx as f32 + 2.0, cy as f32 + 3.0, 1.8, 0xD97706);
            p.fill_rounded_rect(cx + 4, cy + 1, 6, 9, 1, M3_TEAL);
        }
        "app_calc" | "app_calculator" | "calculator" => {
            // 4-Quadrant Calculator
            p.fill_rounded_rect(cx - 11, cy - 11, 22, 22, 6, M3_SURFACE_CONTAINER_HIGH);
            p.draw_rounded_outline(cx - 11, cy - 11, 22, 22, 6, accent);
            // + symbol
            p.draw_line_aa(cx as f32 - 6.0, cy as f32 - 6.0, cx as f32 - 2.0, cy as f32 - 6.0, 1.6, M3_AMBER);
            p.draw_line_aa(cx as f32 - 4.0, cy as f32 - 8.0, cx as f32 - 4.0, cy as f32 - 4.0, 1.6, M3_AMBER);
            // - symbol
            p.draw_line_aa(cx as f32 + 2.0, cy as f32 - 6.0, cx as f32 + 6.0, cy as f32 - 6.0, 1.6, M3_TEAL);
            // × symbol
            p.draw_line_aa(cx as f32 - 6.0, cy as f32 + 3.0, cx as f32 - 2.0, cy as f32 + 7.0, 1.6, M3_GREEN);
            p.draw_line_aa(cx as f32 - 2.0, cy as f32 + 3.0, cx as f32 - 6.0, cy as f32 + 7.0, 1.6, M3_GREEN);
            // = symbol
            p.draw_line_aa(cx as f32 + 2.0, cy as f32 + 4.0, cx as f32 + 6.0, cy as f32 + 4.0, 1.6, M3_PRIMARY);
            p.draw_line_aa(cx as f32 + 2.0, cy as f32 + 6.5, cx as f32 + 6.5, cy as f32 + 6.5, 1.6, M3_PRIMARY);
        }
        "app_music" | "music" | "app_vlc" => {
            // Vinyl Record & Audio Note
            p.fill_circle_aa(cx, cy, 11.0, accent);
            p.fill_circle_aa(cx, cy, 8.0, M3_SURFACE);
            p.fill_circle_aa(cx, cy, 4.5, M3_ORANGE);
            p.fill_circle_aa(cx, cy, 1.5, 0xFFFFFF);
            p.fill_rounded_rect(cx + 7, cy - 4, 2, 8, 1, M3_TEAL);
            p.fill_rounded_rect(cx + 10, cy - 7, 2, 14, 1, M3_TEAL);
        }
        "app_terminal" | "terminal" => {
            // Sleek Terminal Console
            p.fill_rounded_rect(cx - 12, cy - 9, 24, 18, 4, 0x100E14);
            p.draw_rounded_outline(cx - 12, cy - 9, 24, 18, 4, M3_OUTLINE_VARIANT);
            p.fill_circle_aa(cx - 8, cy - 6, 1.5, M3_ERROR);
            p.fill_circle_aa(cx - 4, cy - 6, 1.5, M3_AMBER);
            p.fill_circle_aa(cx, cy - 6, 1.5, M3_GREEN);
            // > prompt
            p.draw_line_aa(cx as f32 - 8.0, cy as f32 - 2.0, cx as f32 - 5.0, cy as f32, 1.5, accent);
            p.draw_line_aa(cx as f32 - 5.0, cy as f32, cx as f32 - 8.0, cy as f32 + 2.0, 1.5, accent);
            // _ cursor
            p.draw_line_aa(cx as f32 - 3.0, cy as f32 + 2.0, cx as f32 + 3.0, cy as f32 + 2.0, 1.8, M3_GREEN);
        }
        "app_softbus" | "softbus" => {
            // SoftBus Mesh Node
            p.draw_line_aa(cx as f32 - 8.0, cy as f32, cx as f32 + 8.0, cy as f32, 2.0, M3_PRIMARY);
            p.draw_line_aa(cx as f32, cy as f32 - 8.0, cx as f32, cy as f32 + 8.0, 2.0, M3_PRIMARY);
            p.fill_circle_aa(cx, cy, 3.5, 0xFFFFFF);
            p.fill_circle_aa(cx - 8, cy, 2.8, M3_TEAL);
            p.fill_circle_aa(cx + 8, cy, 2.8, M3_GREEN);
            p.fill_circle_aa(cx, cy - 8, 2.8, M3_AMBER);
            p.fill_circle_aa(cx, cy + 8, 2.8, M3_PRIMARY);
        }
        "app_nilpkg" | "app_store" | "store" => {
            // App Store Bag with Download Arrow
            p.fill_rounded_rect(cx - 9, cy - 10, 18, 5, 2, accent);
            p.fill_rounded_rect(cx - 7, cy - 8, 14, 3, 1, M3_SURFACE);
            p.fill_rounded_rect(cx - 11, cy - 5, 22, 17, 4, accent);
            p.draw_line_aa(cx as f32, cy as f32 - 2.0, cx as f32, cy as f32 + 4.0, 2.0, 0xFFFFFF);
            p.fill_triangle_aa(
                (cx as f32 - 4.0, cy as f32 + 2.0),
                (cx as f32 + 4.0, cy as f32 + 2.0),
                (cx as f32, cy as f32 + 6.0),
                0xFFFFFF,
            );
        }
        "app_settings" | "settings" => {
            // Smooth 8-Tooth Gear Cog
            p.fill_rounded_rect(cx - 3, cy - 11, 6, 22, 2, accent);
            p.fill_rounded_rect(cx - 11, cy - 3, 22, 6, 2, accent);
            p.fill_circle_aa(cx, cy, 9.0, accent);
            p.fill_circle_aa(cx, cy, 4.0, M3_SURFACE);
            p.fill_circle_aa(cx, cy, 1.5, accent);
        }
        "weather_sun_cloud" => {
            // Crisp Procedural Sun & Fluffy Cloud
            let sun_cx = cx - 4;
            let sun_cy = cy - 4;
            p.fill_circle_aa(sun_cx, sun_cy, 5.0, M3_AMBER);
            p.draw_line_aa(sun_cx as f32, sun_cy as f32 - 8.0, sun_cx as f32, sun_cy as f32 - 6.0, 1.5, M3_AMBER);
            p.draw_line_aa(sun_cx as f32, sun_cy as f32 + 6.0, sun_cx as f32, sun_cy as f32 + 8.0, 1.5, M3_AMBER);
            p.draw_line_aa(sun_cx as f32 - 8.0, sun_cy as f32, sun_cx as f32 - 6.0, sun_cy as f32, 1.5, M3_AMBER);
            p.draw_line_aa(sun_cx as f32 + 6.0, sun_cy as f32, sun_cx as f32 + 8.0, sun_cy as f32, 1.5, M3_AMBER);

            let cl_x = cx - 2;
            let cl_y = cy;
            p.fill_rounded_rect(cl_x - 7, cl_y, 18, 9, 4, 0xEEF2F6);
            p.fill_circle_aa(cl_x, cl_y, 4.8, 0xEEF2F6);
            p.fill_circle_aa(cl_x + 5, cl_y + 1, 3.8, 0xEEF2F6);
        }
        "music_note" => {
            // Clean Beamed Music Notes
            let nx = cx - 6;
            let ny = cy - 7;
            p.fill_circle_aa(nx + 3, ny + 12, 3.0, accent);
            p.fill_circle_aa(nx + 11, ny + 10, 3.0, accent);
            p.draw_line_aa(nx as f32 + 4.5, ny as f32 + 1.0, nx as f32 + 4.5, ny as f32 + 12.0, 1.8, accent);
            p.draw_line_aa(nx as f32 + 12.5, ny as f32 - 1.0, nx as f32 + 12.5, ny as f32 + 10.0, 1.8, accent);
            p.draw_line_aa(nx as f32 + 4.0, ny as f32, nx as f32 + 13.0, ny as f32 - 2.0, 2.5, accent);
        }
        "nav_back" => {
            // Material 3 Smooth Left Chevron ‹
            p.draw_line_aa(cx as f32 + 3.0, cy as f32 - 6.0, cx as f32 - 3.0, cy as f32, 2.2, accent);
            p.draw_line_aa(cx as f32 - 3.0, cy as f32, cx as f32 + 3.0, cy as f32 + 6.0, 2.2, accent);
        }
        "nav_home" => {
            // Material 3 Home House with smooth gable
            p.fill_triangle_aa(
                (cx as f32, cy as f32 - 8.0),
                (cx as f32 - 8.0, cy as f32 - 1.0),
                (cx as f32 + 8.0, cy as f32 - 1.0),
                accent,
            );
            p.fill_rounded_rect(cx - 6, cy - 1, 12, 8, 2, accent);
            p.fill_rect(cx - 2, cy + 2, 4, 5, M3_PRIMARY_CONTAINER);
        }
        "nav_lock" => {
            // Material 3 Padlock
            p.draw_circle_outline_aa(cx, cy - 3, 4.0, 2.0, accent);
            p.fill_rounded_rect(cx - 6, cy - 1, 12, 9, 2, accent);
            p.fill_circle_aa(cx, cy + 3, 1.5, M3_SURFACE_CONTAINER);
        }
        "nav_tasks" | "tasks" | "recents" => {
            // Material 3 Recent Apps / Multitasking Switcher (overlapping cards)
            p.fill_rounded_rect(cx - 7, cy - 7, 10, 10, 2, accent);
            p.fill_rect(cx - 5, cy - 5, 6, 6, M3_SURFACE_CONTAINER);
            p.fill_rounded_rect(cx - 3, cy - 3, 10, 10, 2, accent);
            p.fill_rect(cx - 1, cy - 1, 6, 6, M3_PRIMARY_CONTAINER);
        }
        "flashlight" => {
            p.fill_rounded_rect(cx - 3, cy - 2, 6, 11, 2, accent);
            p.fill_rounded_rect(cx - 6, cy - 7, 12, 6, 2, accent);
            p.fill_rect(cx - 2, cy - 9, 4, 2, M3_AMBER);
        }
        "emergency" => {
            p.fill_rounded_rect(cx - 6, cy - 6, 12, 12, 3, accent);
            p.draw_line_aa(cx as f32 - 4.0, cy as f32, cx as f32 + 4.0, cy as f32, 2.0, 0xFFFFFF);
            p.draw_line_aa(cx as f32, cy as f32 - 4.0, cx as f32, cy as f32 + 4.0, 2.0, 0xFFFFFF);
        }
        "app_android" | _ => {
            // Onuron Orbital Nucleus
            p.draw_circle_outline_aa(cx, cy, 9.5, 1.5, accent);
            p.fill_circle_aa(cx, cy, 4.0, accent);
            p.fill_circle_aa(cx, cy, 1.8, 0xFFFFFF);
            p.fill_circle_aa(cx - 8, cy, 2.0, M3_TEAL);
            p.fill_circle_aa(cx + 8, cy, 2.0, M3_TEAL);
            p.fill_circle_aa(cx, cy - 8, 2.0, M3_GREEN);
            p.fill_circle_aa(cx, cy + 8, 2.0, M3_GREEN);
        }
    }
}

fn render_home(p: &mut FramePainter, _state: &SimState) {
    p.fill_rect(0, 0, p.width as u16, p.height as u16, M3_SURFACE);

    let cx = p.width as i16 / 2;
    let start_y: i16 = 38;

    // 1. Hero Digital Clock in Bengali (M3 Display Large Typography)
    let time_str = get_ist_time_str();
    let clock_w = p.text_width(44.0, &time_str, true);
    p.draw_text_smooth(cx - clock_w / 2 + 1, start_y + 11, 44.0, &time_str, M3_SURFACE_CONTAINER_HIGHEST, true);
    p.draw_text_smooth(cx - clock_w / 2, start_y + 10, 44.0, &time_str, M3_ON_SURFACE, true);

    // 2. Date in Bengali (M3 Primary Lavender Label)
    let date_str = "রবিবার, ৬ সেপ্টেম্বর ২০২৬ (IST)";
    let date_w = p.text_width(12.5, date_str, false);
    let date_y = start_y + 58;
    p.draw_text_smooth(cx - date_w / 2, date_y, 12.5, date_str, M3_PRIMARY, false);

    // 3. Hardware System Chip Card (M3 Assist Chip Pill)
    let chip_y = date_y + 20;
    let chip_w = p.width as u16 - 24;
    p.fill_rounded_rect(12, chip_y, chip_w, 28, 14, M3_SURFACE_CONTAINER_HIGH);
    p.draw_rounded_outline(12, chip_y, chip_w, 28, 14, M3_OUTLINE_VARIANT);
    let chip_text = "Snapdragon 8 Elite • 120Hz AMOLED • Onuron M3";
    let chip_tw = p.text_width(9.5, chip_text, false);
    let bolt_x = cx - chip_tw / 2 - 12;
    let bolt_y = chip_y + 9;
    p.fill_rect(bolt_x + 2, bolt_y, 3, 3, M3_AMBER);
    p.fill_rect(bolt_x, bolt_y + 2, 5, 2, M3_AMBER);
    p.fill_rect(bolt_x + 1, bolt_y + 4, 3, 3, M3_AMBER);
    p.draw_text_smooth(cx - chip_tw / 2, chip_y + 17, 9.5, chip_text, M3_ON_SURFACE_VARIANT, false);

    // 4. Full 12-App Grid (3 rows x 4 columns) with Material You Squircles
    let grid_y = chip_y + 36;
    let icon_size: u16 = 50;
    let col_w = (p.width as i16 - 24) / 4;

    let apps: &[(&str, u32, &str)] = &[
        ("ফোন", M3_GREEN, "app_phone"),
        ("বার্তা", M3_AMBER, "app_messages"),
        ("ব্রাউজার", M3_TEAL, "app_browser"),
        ("ফাইল", M3_BLUE, "app_files"),
        ("নোটস", M3_AMBER, "app_notes"),
        ("ক্যালক", M3_PRIMARY, "app_calc"),
        ("মিউজিক", M3_ORANGE, "app_music"),
        ("টার্মিনাল", M3_TEAL, "app_terminal"),
        ("সফটবাস", M3_PRIMARY, "app_softbus"),
        ("স্টোর", M3_PINK, "app_nilpkg"),
        ("সেটিংস", M3_SECONDARY, "app_settings"),
        ("অনুরণ", M3_TEAL, "app_android"),
    ];

    for (i, (title, accent, id)) in apps.iter().enumerate() {
        let r = (i / 4) as i16;
        let c = (i % 4) as i16;
        let icon_cx = 12 + c * col_w + col_w / 2;
        let icon_cy = grid_y + r * 78;

        let ix = icon_cx - (icon_size as i16) / 2;
        // Material You Container Squircle (radius: 16)
        let container_bg = blend_pixel(M3_SURFACE_CONTAINER_HIGH, *accent, 36);
        p.fill_rounded_rect(ix, icon_cy, icon_size, icon_size, 16, container_bg);
        p.draw_rounded_outline(ix, icon_cy, icon_size, icon_size, 16, blend_pixel(M3_OUTLINE_VARIANT, *accent, 60));

        // Vector Icon
        draw_vector_icon(p, icon_cx, icon_cy + 25, id, *accent, 24);

        // Label below
        let tw = p.text_width(11.0, title, false);
        p.draw_text_smooth(icon_cx - tw / 2, icon_cy + (icon_size as i16) + 14, 11.0, title, M3_ON_SURFACE, false);

        p.register_button(ix, icon_cy, icon_size, icon_size + 18, id);
    }

    // 5. Glance Widgets Cards (Material 3 Surface Container Low with zero missing character boxes)
    let widget_y = grid_y + 3 * 78 + 10;
    let card_w = (p.width as u16 - 24) / 2;

    // Weather widget card
    let wx = 12i16;
    p.fill_rounded_rect(wx, widget_y, card_w - 4, 66, 16, M3_SURFACE_CONTAINER_LOW);
    p.draw_rounded_outline(wx, widget_y, card_w - 4, 66, 16, M3_OUTLINE_VARIANT);
    draw_vector_icon(p, wx + 18, widget_y + 18, "weather_sun_cloud", M3_AMBER, 16);
    p.draw_text_smooth(wx + 34, widget_y + 11, 12.0, "শিলিগুড়ি", M3_ON_SURFACE, false);
    p.draw_text_smooth(wx + 10, widget_y + 32, 17.0, "২৬°C", M3_PRIMARY, true);
    p.draw_text_smooth(wx + 10, widget_y + 50, 10.5, "আংশিক মেঘলা", M3_ON_SURFACE_VARIANT, false);
    p.register_button(wx, widget_y, card_w - 4, 66, "home_weather_card");

    // Music glance card
    let mx = 12 + card_w as i16 + 4;
    p.fill_rounded_rect(mx, widget_y, card_w - 4, 66, 16, M3_SURFACE_CONTAINER_LOW);
    p.draw_rounded_outline(mx, widget_y, card_w - 4, 66, 16, M3_OUTLINE_VARIANT);
    draw_vector_icon(p, mx + 18, widget_y + 18, "music_note", M3_TERTIARY, 16);
    p.draw_text_smooth(mx + 34, widget_y + 11, 12.0, "এখন বাজছে", M3_TERTIARY, false);
    p.draw_text_smooth(mx + 10, widget_y + 32, 12.5, "আনন্দলোকে...", M3_ON_SURFACE, false);
    p.draw_text_smooth(mx + 10, widget_y + 50, 10.5, "রবীন্দ্রনাথ ঠাকুর", M3_ON_SURFACE_VARIANT, false);
    p.register_button(mx, widget_y, card_w - 4, 66, "app_music");

    // RAM / Storage status strip
    let ram_y = widget_y + 72;
    let ram_w = p.width as u16 - 24;
    p.fill_rounded_rect(12, ram_y, ram_w, 24, 12, M3_SURFACE_CONTAINER_LOW);
    p.draw_rounded_outline(12, ram_y, ram_w, 24, 12, M3_OUTLINE_VARIANT);
    let ram_str = "RAM: ৩.৪/১৬ GB সক্রিয় • স্টোরেজ: ১২৮ GB ফাঁকা";
    let ram_tw = p.text_width(10.0, ram_str, false);
    p.draw_text_smooth(cx - ram_tw / 2, ram_y + 15, 10.0, ram_str, M3_GREEN, false);
}

fn render_status_bar(p: &mut FramePainter, state: &SimState) {
    p.fill_rect(0, 0, p.width as u16, 32, M3_SURFACE);

    // 1. Time (Left) in Bengali IST
    let time_str = get_ist_time_str();
    p.draw_text_smooth(12, 9, 13.0, &time_str, M3_ON_SURFACE, true);

    // 2. Dynamic Island (Center)
    let center_x = p.width as i16 / 2;
    p.fill_rounded_rect(center_x - 38, 4, 76, 22, 11, 0x000000);
    p.fill_rect(center_x - 3, 11, 6, 8, 0x181F2E);
    p.fill_rounded_rect(center_x + 8, 12, 6, 6, 3, 0x1C2740);
    p.fill_rounded_rect(center_x - 14, 12, 6, 6, 3, 0x1C2740);
    p.register_button(center_x - 38, 4, 76, 22, "toggle_island");

    // 3. Dynamic Wi-Fi (Left of Cellular)
    let wifi_x: i16 = 212;
    if state.wifi_enabled {
        let c_top = if state.wifi_signal_level >= 3 { M3_TEAL } else { 0x334155 };
        let c_mid = if state.wifi_signal_level >= 2 { M3_TEAL } else { 0x334155 };
        let c_dot = if state.wifi_signal_level >= 1 { M3_TEAL } else { 0x334155 };

        p.fill_rounded_rect(wifi_x, 10, 14, 2, 1, c_top);
        p.fill_rounded_rect(wifi_x + 2, 14, 10, 2, 1, c_mid);
        p.fill_rounded_rect(wifi_x + 5, 18, 4, 3, 1, c_dot);
    } else {
        p.fill_rounded_rect(wifi_x, 10, 14, 2, 1, 0x334155);
        p.draw_text_smooth(wifi_x + 3, 8, 10.0, "✕", M3_ON_SURFACE_VARIANT, false);
    }
    p.register_button(wifi_x - 4, 4, 22, 24, "toggle_wifi");

    // 4. Cellular Network Generation Badge (৫G / ৪G / ৩G / E / অফ)
    let net_x: i16 = 232;
    if state.cellular_enabled {
        let (badge, color) = match state.cellular_signal_level {
            4 | 3 => ("৫G", M3_TEAL),
            2 => ("৪G", M3_BLUE),
            1 => ("৩G", M3_AMBER),
            _ => ("E", M3_ERROR),
        };
        p.draw_text_smooth(net_x, 9, 11.5, badge, color, true);
    } else {
        p.draw_text_smooth(net_x, 10, 10.0, "অফ", M3_ON_SURFACE_VARIANT, false);
    }

    // 5. Cellular Signal Towers (4 Vertical Bars)
    let sig_x: i16 = 256;
    let tower_base_y: i16 = 21;
    let bar_heights = [3i16, 5, 7, 10];

    if state.cellular_enabled {
        for (i, h) in bar_heights.iter().enumerate() {
            let bx = sig_x + i as i16 * 4;
            let by = tower_base_y - *h;
            let col = if (i as u8) < state.cellular_signal_level {
                M3_ON_SURFACE
            } else {
                0x334155
            };
            p.fill_rect(bx, by, 2, *h as u16, col);
        }
    } else {
        for (i, h) in bar_heights.iter().enumerate() {
            let bx = sig_x + i as i16 * 4;
            let by = tower_base_y - *h;
            p.fill_rect(bx, by, 2, *h as u16, 0x334155);
        }
        p.draw_text_smooth(sig_x + 3, 8, 10.0, "✕", M3_ERROR, false);
    }
    p.register_button(net_x - 2, 4, 44, 24, "cycle_signal");

    // 6. Dynamic Battery
    let bat_x: i16 = 282;
    p.fill_rounded_rect(bat_x, 10, 24, 12, 3, M3_SURFACE_CONTAINER_HIGH);
    p.draw_rounded_outline(bat_x, 10, 24, 12, 3, M3_OUTLINE_VARIANT);
    p.fill_rect(bat_x + 24, 13, 2, 6, M3_OUTLINE_VARIANT);

    let fill_w = ((state.battery_level as i16 * 18) / 100).max(2).min(18);
    let fill_color = if state.battery_level <= 20 {
        M3_ERROR
    } else if state.battery_level <= 40 {
        M3_AMBER
    } else {
        M3_GREEN
    };
    p.fill_rounded_rect(bat_x + 2, 12, fill_w as u16, 8, 1, fill_color);

    if state.battery_charging {
        let bx = bat_x + 10;
        let by = 13;
        p.fill_rect(bx + 1, by, 3, 2, M3_AMBER);
        p.fill_rect(bx - 1, by + 2, 5, 2, M3_AMBER);
        p.fill_rect(bx, by + 4, 3, 2, M3_AMBER);
    }
    p.register_button(bat_x - 2, 4, 32, 24, "toggle_battery_charge");

    // Hairline bottom border
    p.fill_rect(0, 32, p.width as u16, 1, M3_OUTLINE_VARIANT);
}

fn render_bottom_nav(p: &mut FramePainter, current: &Screen) {
    let nav_h = 56u16;
    let nav_y = (p.height as i16) - nav_h as i16;

    // Material 3 Navigation Bar Background
    p.fill_rect(0, nav_y, p.width as u16, nav_h, M3_SURFACE_CONTAINER);
    p.fill_rect(0, nav_y, p.width as u16, 1, M3_OUTLINE_VARIANT);

    let item_w = (p.width as i16) / 4;
    let items = [
        ("nav_back", "ব্যাক", "nav_back"),
        ("nav_home", "হোম", "nav_home"),
        ("nav_tasks", "টাস্ক", "nav_tasks"),
        ("nav_lock", "লক", "nav_lock"),
    ];

    for (i, (id, label, icon_type)) in items.iter().enumerate() {
        let ix = i as i16 * item_w;
        let is_active = match (i, current) {
            (0, _) => false,
            (1, Screen::Home) => true,
            (2, Screen::TaskSwitcher) => true,
            (3, Screen::Lockscreen) => true,
            _ => false,
        };

        let pill_w = 48u16;
        let pill_h = 26u16;
        let pill_x = ix + (item_w - pill_w as i16) / 2;
        let pill_y = nav_y + 6;

        if is_active {
            // M3 Active Pill Indicator
            p.fill_pill(pill_x, pill_y, pill_w, pill_h, M3_PRIMARY_CONTAINER);
        }

        // Vector Icon
        let icon_col = if is_active { M3_ON_PRIMARY_CONTAINER } else { M3_ON_SURFACE_VARIANT };
        draw_vector_icon(p, ix + item_w / 2, pill_y + 13, icon_type, icon_col, 16);

        // Label below
        let fg = if is_active { M3_PRIMARY } else { M3_ON_SURFACE_VARIANT };
        let tw = p.text_width(10.5, label, false);
        p.draw_text_smooth(ix + (item_w - tw) / 2, nav_y + 38, 10.5, label, fg, false);

        p.register_button(ix, nav_y, item_w as u16, nav_h, id);
    }
}

fn render_lockscreen(p: &mut FramePainter, state: &SimState) {
    let center_x = p.width as i16 / 2;
    let time_str = get_ist_time_str();

    // M3 Atmospheric ambient top glow
    p.fill_rect(0, 0, p.width as u16, p.height as u16, M3_SURFACE);

    // M3 Display Large Clock
    let tw = p.text_width(52.0, &time_str, true);
    p.draw_text_smooth(center_x - tw / 2, 58, 52.0, &time_str, M3_ON_SURFACE, true);

    // Date below in M3 Primary Lilac
    let date_str = "রবিবার, ৬ সেপ্টেম্বর ২০২৬ (IST)";
    let dw = p.text_width(12.5, date_str, false);
    p.draw_text_smooth(center_x - dw / 2, 118, 12.5, date_str, M3_PRIMARY, false);

    // User avatar pill
    let user_label = format!("👤 {}", state.user_name);
    let uw = p.text_width(13.5, &user_label, false);
    let up_x = center_x - uw / 2 - 12;
    p.fill_pill(up_x, 142, (uw + 24) as u16, 28, M3_SURFACE_CONTAINER_HIGH);
    p.draw_rounded_outline(up_x, 142, (uw + 24) as u16, 28, 14, M3_OUTLINE_VARIANT);
    p.draw_text_smooth(up_x + 12, 149, 13.5, &user_label, M3_ON_SURFACE, false);

    // PIN dots — M3 glowing circles
    let dot_y = 182;
    let dot_start_x = center_x - 54;
    for i in 0..4usize {
        let dx = dot_start_x + (i as i16 * 30);
        if state.pin_input.len() > i {
            p.fill_rounded_rect(dx, dot_y, 16, 16, 8, M3_PRIMARY);
            p.fill_rounded_rect(dx + 3, dot_y + 3, 10, 10, 5, 0xFFFFFF);
        } else {
            p.fill_rounded_rect(dx, dot_y, 16, 16, 8, M3_SURFACE_CONTAINER_HIGHEST);
            p.draw_rounded_outline(dx, dot_y, 16, 16, 8, M3_OUTLINE_VARIANT);
        }
    }

    if state.lock_error {
        let err_msg = "⚠ ভুল পিন নম্বর — ডিফল্ট: 1234";
        let ew = p.text_width(12.5, err_msg, false);
        p.draw_text_smooth(center_x - ew / 2, 208, 12.5, err_msg, M3_ERROR, false);
    } else {
        let hint_msg = "কীপ্যাডে চাপুন অথবা কীবোর্ডে 1234 লিখুন";
        let hw = p.text_width(11.5, hint_msg, false);
        p.draw_text_smooth(center_x - hw / 2, 208, 11.5, hint_msg, M3_ON_SURFACE_VARIANT, false);
    }

    // Keypad — M3 circular style
    let pad_x = (p.width as i16 - 252) / 2;
    let pad_y = 232;
    let btn_size: u16 = 68;
    let gap: i16 = 10;

    let keys = [
        ["১", "২", "৩"],
        ["৪", "৫", "৬"],
        ["৭", "৮", "৯"],
        ["<", "০", "আনলক"],
    ];

    for (r, row) in keys.iter().enumerate() {
        for (c, &label) in row.iter().enumerate() {
            let bx = pad_x + c as i16 * (btn_size as i16 + gap);
            let by = pad_y + r as i16 * (btn_size as i16 + gap);
            let id = format!("lock_key_{}", label);
            let (bg, fg) = if label == "আনলক" {
                (M3_PRIMARY_CONTAINER, M3_ON_PRIMARY_CONTAINER)
            } else if label == "<" {
                (M3_SURFACE_CONTAINER_HIGHEST, M3_AMBER)
            } else {
                (M3_SURFACE_CONTAINER_HIGH, M3_ON_SURFACE)
            };
            p.fill_rounded_rect(bx, by, btn_size, btn_size, (btn_size / 2) as i16, bg);
            p.draw_rounded_outline(bx, by, btn_size, btn_size, (btn_size / 2) as i16, M3_OUTLINE_VARIANT);
            let tw = p.text_width(16.0, label, false);
            p.draw_text_smooth(bx + (btn_size as i16 - tw) / 2, by + (btn_size as i16 - 16) / 2, 16.0, label, fg, false);
            p.register_button(bx, by, btn_size, btn_size, &id);
        }
    }

    // Quick Bottom Shortcuts (Flashlight & Emergency Call)
    let bottom_y = (p.height as i16) - 62;
    // Torch button on bottom-left
    let torch_bg = if state.torch_enabled { M3_AMBER } else { M3_SURFACE_CONTAINER_HIGH };
    p.fill_rounded_rect(24, bottom_y, 48, 48, 24, torch_bg);
    p.draw_rounded_outline(24, bottom_y, 48, 48, 24, M3_OUTLINE_VARIANT);
    draw_vector_icon(p, 48, bottom_y + 24, "flashlight", if state.torch_enabled { M3_SURFACE } else { M3_AMBER }, 20);
    p.register_button(24, bottom_y, 48, 48, "lock_torch");

    // Emergency Call button on bottom-right
    let emer_x = (p.width as i16) - 72;
    p.fill_rounded_rect(emer_x, bottom_y, 48, 48, 24, M3_SURFACE_CONTAINER_HIGH);
    p.draw_rounded_outline(emer_x, bottom_y, 48, 48, 24, M3_OUTLINE_VARIANT);
    draw_vector_icon(p, emer_x + 24, bottom_y + 24, "emergency", M3_GREEN, 20);
    p.register_button(emer_x, bottom_y, 48, 48, "lock_emergency");
}

fn render_app_browser(p: &mut FramePainter, state: &SimState) {
    let pw = p.width as i16;

    // 1. M3 Top Header
    p.fill_rect(0, 32, p.width as u16, 36, M3_SURFACE_CONTAINER);
    p.draw_text_smooth(12, 41, 15.0, "নীলজার ব্রাউজার (Chromium V8)", M3_TEAL, false);

    // 2. M3 Address & Control Bar
    p.fill_rect(0, 68, p.width as u16, 36, M3_SURFACE_CONTAINER);
    p.draw_button(6, 72, 28, 28, "<", M3_SURFACE_CONTAINER_HIGH, M3_ON_SURFACE, "browser_back");
    p.draw_button(36, 72, 28, 28, "↻", M3_SURFACE_CONTAINER_HIGH, M3_ON_SURFACE, "browser_reload");

    let url_x = 68i16;
    let url_w = (pw - url_x - 42) as u16;
    p.fill_pill(url_x, 72, url_w, 28, M3_SURFACE_CONTAINER_HIGH);
    p.draw_rounded_outline(url_x, 72, url_w, 28, 14, M3_OUTLINE_VARIANT);

    let display_url = if state.browser_is_editing_url {
        let mut s = state.browser_url_input.clone();
        if state.term_cursor_ticks % 60 < 30 {
            s.push('|');
        }
        s
    } else {
        state.browser_url.clone()
    };
    let short_url: String = display_url.chars().take(26).collect();
    p.draw_text_smooth(url_x + 10, 78, 11.5, &short_url, M3_ON_SURFACE, false);
    p.register_button(url_x, 72, url_w, 28, "browser_url_bar");

    p.draw_button(pw - 34, 72, 28, 28, "Go", M3_PRIMARY_CONTAINER, M3_ON_PRIMARY_CONTAINER, "browser_go");

    // 3. M3 Tab Strip Bar
    p.fill_rect(0, 104, p.width as u16, 28, M3_SURFACE_CONTAINER_LOW);
    p.fill_rect(0, 104, p.width as u16, 1, M3_OUTLINE_VARIANT);

    let mut tab_x = 6i16;
    for (i, tab) in state.browser_tabs.iter().enumerate() {
        let is_active = i == state.browser_active_tab;
        let tab_w = 78u16;
        let bg = if is_active { M3_PRIMARY_CONTAINER } else { M3_SURFACE_CONTAINER_HIGH };
        let fg = if is_active { M3_ON_PRIMARY_CONTAINER } else { M3_ON_SURFACE_VARIANT };

        p.fill_rounded_rect(tab_x, 106, tab_w, 22, 6, bg);
        let title_disp: String = tab.title.chars().take(6).collect();
        p.draw_text_smooth(tab_x + 6, 111, 10.5, &title_disp, fg, false);

        // Close 'x' button
        p.draw_text_smooth(tab_x + tab_w as i16 - 16, 111, 10.5, "x", fg, false);
        p.register_button(tab_x + tab_w as i16 - 20, 106, 20, 22, &format!("tab_close_{}", i));

        p.register_button(tab_x, 106, tab_w - 20, 22, &format!("tab_select_{}", i));
        tab_x += tab_w as i16 + 4;
        if tab_x > pw - 36 { break; }
    }

    // New Tab (+) Button
    p.draw_button(pw - 28, 106, 22, 22, "+", M3_SURFACE_CONTAINER_HIGH, M3_TEAL, "tab_add");

    // 4. WebView2 Container Outline
    p.draw_rounded_outline(3, 134, 324, 492, 12, M3_OUTLINE_VARIANT);
}

fn render_app_notes(p: &mut FramePainter, state: &SimState) {
    let pw = p.width as i16;

    if state.notes_editing {
        // M3 Note Editor Header
        p.fill_rect(0, 32, p.width as u16, 40, M3_SURFACE_CONTAINER);
        p.draw_text_smooth(16, 43, 17.0, "নোট সম্পাদনা (ArkTS Notes)", M3_AMBER, false);

        let btn_w = (p.width - 24) as u16;
        let half_w = (btn_w / 2) - 4;
        p.draw_button(12, 80, half_w, 34, "✓ সংরক্ষণ", M3_PRIMARY_CONTAINER, M3_ON_PRIMARY_CONTAINER, "notes_save");
        p.draw_button(12 + half_w as i16 + 8, 80, half_w, 34, "✕ বাতিল", M3_SURFACE_CONTAINER_HIGH, M3_ON_SURFACE, "notes_cancel");

        // Action Sub-bar: Pin & Delete
        let sub_y = 120i16;
        if state.notes_edit_id.is_some() {
            let is_pinned = state.notes.iter().find(|n| Some(n.id) == state.notes_edit_id).map(|n| n.pinned).unwrap_or(false);
            let pin_lbl = if is_pinned { "📌 পিন সরানো" } else { "📌 পিন করুন" };
            p.draw_button(12, sub_y, half_w, 28, pin_lbl, M3_SECONDARY_CONTAINER, M3_ON_SECONDARY_CONTAINER, "note_toggle_pin");
            p.draw_button(12 + half_w as i16 + 8, sub_y, half_w, 28, "🗑 মুছুন", M3_ERROR_CONTAINER, M3_ON_ERROR_CONTAINER, "note_delete");
        }

        // Title box
        let title_y = if state.notes_edit_id.is_some() { 156 } else { 124 };
        p.fill_rounded_rect(12, title_y, btn_w, 38, 12, M3_SURFACE_CONTAINER_HIGH);
        p.draw_rounded_outline(12, title_y, btn_w, 38, 12, M3_OUTLINE_VARIANT);
        let title_disp = if state.notes_edit_title.is_empty() { "নোটের শিরোনাম লিখুন..." } else { &state.notes_edit_title };
        let title_fg = if state.notes_edit_title.is_empty() { M3_ON_SURFACE_VARIANT } else { M3_ON_SURFACE };
        p.draw_text_smooth(20, title_y + 10, 14.5, title_disp, title_fg, false);

        // Content Area
        let content_y = title_y + 46;
        let content_h = ((p.height as i16) - 60 - content_y).max(160) as u16;
        p.fill_rounded_rect(12, content_y, btn_w, content_h, 14, M3_SURFACE_CONTAINER_LOW);
        p.draw_rounded_outline(12, content_y, btn_w, content_h, 14, M3_OUTLINE_VARIANT);

        let max_text_w = (btn_w as i16) - 20;
        let cursor_char = if (state.term_cursor_ticks / 25) % 2 == 0 { "█" } else { " " };
        let chars: Vec<char> = state.notes_edit_content.chars().collect();
        let cur = state.notes_cursor_pos.min(chars.len());
        let before: String = chars[..cur].iter().collect();
        let after: String = chars[cur..].iter().collect();
        let full_text_with_cursor = format!("{}{}{}", before, cursor_char, after);

        let wrapped_lines = wrap_text_to_lines(p, &full_text_with_cursor, max_text_w, 13.5);
        for (i, line) in wrapped_lines.iter().enumerate().take(18) {
            let ly = content_y + 12 + (i as i16 * 22);
            p.draw_text_smooth(20, ly, 13.5, line, M3_ON_SURFACE, false);
        }
        return;
    }

    // Notes List Screen
    p.fill_rect(0, 32, p.width as u16, 40, M3_SURFACE_CONTAINER);
    p.draw_text_smooth(16, 43, 17.0, "স্মার্ট নোটস (Material Notes)", M3_AMBER, false);

    let btn_x = pw - 96;
    p.draw_button(btn_x, 38, 84, 28, "+ নতুন নোট", M3_PRIMARY_CONTAINER, M3_ON_PRIMARY_CONTAINER, "notes_new");

    // M3 Category Filter Chips
    let chip_w = ((p.width - 36) / 4) as u16;
    let chips = ["সব নোট", "কাজের নোট", "কোড ও আইডিয়া", "ব্যক্তিগত"];
    for (i, &cat) in chips.iter().enumerate() {
        let cx = 12 + (i as i16 * (chip_w as i16 + 4));
        let is_sel = state.notes_category == cat;
        let bg = if is_sel { M3_SECONDARY_CONTAINER } else { M3_SURFACE_CONTAINER_LOW };
        let fg = if is_sel { M3_ON_SECONDARY_CONTAINER } else { M3_ON_SURFACE_VARIANT };
        p.draw_button(cx, 80, chip_w, 28, cat, bg, fg, &format!("notes_cat_{}", cat));
    }

    let list_w = (p.width - 24) as u16;
    let mut list_y = 118;

    for note in state.notes.iter() {
        if state.notes_category != "সব নোট" && note.category != state.notes_category {
            continue;
        }

        p.fill_rounded_rect(12, list_y, list_w, 72, 16, M3_SURFACE_CONTAINER_LOW);
        p.draw_rounded_outline(12, list_y, list_w, 72, 16, M3_OUTLINE_VARIANT);
        p.fill_rounded_rect(12, list_y, 4, 72, 2, note.color);

        let pin_icon = if note.pinned { "📌 " } else { "" };
        let full_title = format!("{}{}", pin_icon, note.title);
        p.draw_text_smooth(24, list_y + 10, 14.5, &full_title, M3_ON_SURFACE, false);

        let snippet = note.content.lines().next().unwrap_or("");
        let snip_disp = safe_truncate(snippet, 30);
        p.draw_text_smooth(24, list_y + 30, 12.0, &snip_disp, M3_ON_SURFACE_VARIANT, false);

        p.draw_text_smooth(24, list_y + 50, 11.0, &note.category, note.color, false);
        p.draw_text_smooth(pw - 60, list_y + 50, 10.5, &note.updated, M3_ON_SURFACE_VARIANT, false);

        let id = format!("note_open_{}", note.id);
        p.register_button(12, list_y, list_w, 72, &id);

        list_y += 80;
        if list_y > (p.height as i16 - 70) {
            break;
        }
    }
}

fn render_app_calculator(p: &mut FramePainter, state: &SimState) {
    let pw = p.width as i16;

    // Header
    p.fill_rect(0, 32, p.width as u16, 40, M3_SURFACE_CONTAINER);
    p.draw_text_smooth(16, 43, 17.0, "ক্যালকুলেটর (Material Calculator)", M3_PRIMARY, false);

    // M3 Display Window with calculation history
    let disp_w = (p.width - 24) as u16;
    p.fill_rounded_rect(12, 78, disp_w, 102, 20, M3_SURFACE_CONTAINER_HIGH);
    p.draw_rounded_outline(12, 78, disp_w, 102, 20, M3_OUTLINE_VARIANT);

    // History roll (previous calculations)
    let mut hist_y = 86i16;
    for (h_expr, h_res) in state.calc_history.iter().rev().take(1) {
        let hist_str = format!("{} = {}", h_expr, h_res);
        let hw = p.text_width(12.0, &hist_str, false);
        p.draw_text_smooth(pw - 24 - hw, hist_y, 12.0, &hist_str, M3_ON_SURFACE_VARIANT, false);
        hist_y += 18;
    }

    // Active Expression line
    let expr_w = p.text_width(26.0, &state.calc_expr, false);
    p.draw_text_smooth(pw - 24 - expr_w, 108, 26.0, &state.calc_expr, M3_TEAL, false);

    // Live Result line
    if !state.calc_result.is_empty() {
        let res_str = format!("= {}", state.calc_result);
        let res_w = p.text_width(22.0, &res_str, false);
        p.draw_text_smooth(pw - 24 - res_w, 148, 22.0, &res_str, M3_GREEN, false);
    }

    // Keypad grid: 5 rows x 4 cols with M3 pill keys
    let pad_x: i16 = 12;
    let pad_y: i16 = 188;
    let btn_w: u16 = 68;
    let btn_h: u16 = 48;
    let gap_x: i16 = 11;
    let gap_y: i16 = 8;

    let keys = [
        ["C", "( )", "%", "÷"],
        ["৭", "৮", "৯", "×"],
        ["৪", "৫", "৬", "-"],
        ["১", "২", "৩", "+"],
        ["±", "০", ".", "="],
    ];

    for (r, row) in keys.iter().enumerate() {
        for (c, &label) in row.iter().enumerate() {
            let bx = pad_x + c as i16 * (btn_w as i16 + gap_x);
            let by = pad_y + r as i16 * (btn_h as i16 + gap_y);
            let id = format!("calc_key_{}", label);

            let (bg, fg) = if label == "=" {
                (M3_PRIMARY_CONTAINER, M3_ON_PRIMARY_CONTAINER)
            } else if label == "÷" || label == "×" || label == "-" || label == "+" {
                (M3_SECONDARY_CONTAINER, M3_ON_SECONDARY_CONTAINER)
            } else if label == "C" {
                (M3_ERROR_CONTAINER, M3_ON_ERROR_CONTAINER)
            } else if label == "%" || label == "( )" || label == "±" {
                (M3_SURFACE_CONTAINER_HIGHEST, M3_AMBER)
            } else {
                (M3_SURFACE_CONTAINER_HIGH, M3_ON_SURFACE)
            };

            p.draw_button(bx, by, btn_w, btn_h, label, bg, fg, &id);
        }
    }
}

fn render_app_music(p: &mut FramePainter, state: &SimState, spectrum: &[u8; 18]) {
    let pw = p.width as i16;
    let cx = pw / 2;

    // 1. Header
    p.fill_rect(0, 32, p.width as u16, 40, M3_SURFACE_CONTAINER);
    p.draw_text_smooth(16, 43, 17.0, "মিউজিক প্লেয়ার (Material Media)", M3_PRIMARY, false);

    let current_track = &MUSIC_TRACKS[state.music_track_idx % MUSIC_TRACKS.len()];

    // 2. Animated Vinyl Record Album Art (Subpixel Anti-Aliased Circles with Glow)
    let vinyl_y: i16 = 80;
    let vinyl_r: f32 = 46.0;
    let v_cx = cx - 8;
    let v_cy = vinyl_y + 46;

    // Ambient radial glow
    let glow_col = blend_pixel(M3_SURFACE, current_track.color, if state.music_playing { 50 } else { 20 });
    p.fill_circle_aa(v_cx, v_cy, vinyl_r + 4.0, glow_col);
    // Vinyl body
    p.fill_circle_aa(v_cx, v_cy, vinyl_r, M3_SURFACE_CONTAINER_LOWEST);
    // Concentric grooves
    p.draw_circle_outline_aa(v_cx, v_cy, 38.0, 1.0, M3_SURFACE_CONTAINER);
    p.draw_circle_outline_aa(v_cx, v_cy, 30.0, 1.0, M3_SURFACE_CONTAINER_HIGH);
    p.draw_circle_outline_aa(v_cx, v_cy, 22.0, 1.0, M3_SURFACE_CONTAINER_HIGHEST);

    // Rotating spindle spokes
    let angle = (state.music_tick as f32 * 0.06).sin();
    let dx = (angle * 12.0) as i16;
    let dy = ((1.0 - angle * angle).max(0.0).sqrt() * 12.0) as i16;
    p.fill_rect(v_cx + dx - 1, v_cy + dy - 1, 3, 3, M3_PRIMARY);
    p.fill_rect(v_cx - dx - 1, v_cy - dy - 1, 3, 3, M3_PRIMARY);

    // Center label & spindle hole
    p.fill_circle_aa(v_cx, v_cy, 15.0, current_track.color);
    p.fill_circle_aa(v_cx, v_cy, 3.5, M3_SURFACE);

    // Tonearm / Needle tracking playback position
    let arm_pivot_x = v_cx + 56;
    let arm_pivot_y = v_cy - 44;
    p.fill_circle_aa(arm_pivot_x, arm_pivot_y, 4.5, M3_OUTLINE);
    p.fill_circle_aa(arm_pivot_x, arm_pivot_y, 2.0, 0xFFFFFF);

    let arm_prog = if state.music_playing { state.music_progress } else { 0.0 };
    let needle_x = v_cx + 26 - (arm_prog * 20.0) as i16;
    let needle_y = v_cy - 16 + (arm_prog * 12.0) as i16;
    p.fill_rect(needle_x, needle_y, 4, 4, M3_PRIMARY);
    p.fill_circle_aa(needle_x + 2, needle_y + 2, 1.5, 0xFFFFFF);

    // 3. Track Info (Title & Artist)
    let tw = p.text_width(16.0, current_track.title, false);
    p.draw_text_smooth(cx - tw / 2, vinyl_y + 100, 16.0, current_track.title, M3_ON_SURFACE, false);

    let aw = p.text_width(12.0, current_track.artist, false);
    p.draw_text_smooth(cx - aw / 2, vinyl_y + 118, 12.0, current_track.artist, M3_ON_SURFACE_VARIANT, false);

    // 4. Live Audio Spectrum Equalizer (Live 18-Band Dynamic Amplitude from Synth)
    let eq_y = vinyl_y + 134;
    let num_bars = 18;
    let bar_w = 9i16;
    let bar_gap = 4i16;
    let total_eq_w = num_bars * (bar_w + bar_gap) - bar_gap;
    let eq_start_x = cx - total_eq_w / 2;

    for i in 0..num_bars {
        let bx = eq_start_x + i * (bar_w + bar_gap);
        let live_val = spectrum[i as usize];
        let h = if state.music_playing {
            (live_val as u16).max(3)
        } else {
            3u16
        };
        let by = eq_y + 24 - (h as i16);
        let col = if i % 2 == 0 { current_track.color } else { M3_PRIMARY };
        p.fill_rounded_rect(bx, by, bar_w as u16, h, 2, col);
    }

    // 5. M3 Scrubber Bar (Interactive touch area)
    let scrub_y = eq_y + 30;
    let scrub_w = p.width as u16 - 32;
    p.fill_pill(16, scrub_y, scrub_w, 8, M3_SURFACE_CONTAINER_HIGH);
    let filled_w = ((scrub_w as f32 * state.music_progress) as u16).max(6).min(scrub_w);
    p.fill_pill(16, scrub_y, filled_w, 8, M3_PRIMARY);
    p.fill_circle_aa(16 + filled_w as i16, scrub_y + 4, 6.0, 0xFFFFFF);
    p.register_button(16, scrub_y - 8, scrub_w, 24, "music_scrub");

    // Bengali Time Labels
    let cur_m = state.music_current_sec / 60;
    let cur_s = state.music_current_sec % 60;
    let cur_time_str = to_bengali_digits(&format!("{:02}:{:02}", cur_m, cur_s));
    p.draw_text_smooth(16, scrub_y + 14, 11.0, &cur_time_str, M3_ON_SURFACE_VARIANT, true);
    p.draw_text_smooth(pw - 48, scrub_y + 14, 11.0, current_track.duration, M3_ON_SURFACE_VARIANT, true);

    // 6. Playback & Volume Controls (Crisp Vector SVG Icons — Zero □ Boxes)
    let ctrl_y = scrub_y + 26;

    // Vol Down
    p.draw_button_with_icon(16, ctrl_y + 4, 34, 34, "media_vol_down", M3_SURFACE_CONTAINER_HIGH, M3_ON_SURFACE_VARIANT, "music_vol_down");

    // Prev Button
    p.draw_button_with_icon(cx - 76, ctrl_y + 2, 38, 38, "media_prev", M3_SURFACE_CONTAINER_HIGH, M3_PRIMARY, "music_prev");

    // Play/Pause Big M3 FAB
    p.fill_rounded_rect(cx - 24, ctrl_y - 2, 48, 48, 24, M3_PRIMARY_CONTAINER);
    p.draw_rounded_outline(cx - 24, ctrl_y - 2, 48, 48, 24, M3_PRIMARY);
    let play_icon = if state.music_playing { "media_pause" } else { "media_play" };
    draw_vector_icon(p, cx, ctrl_y + 22, play_icon, M3_ON_PRIMARY_CONTAINER, 20);
    p.register_button(cx - 24, ctrl_y - 2, 48, 48, "music_toggle_play");

    // Next Button
    p.draw_button_with_icon(cx + 38, ctrl_y + 2, 38, 38, "media_next", M3_SURFACE_CONTAINER_HIGH, M3_PRIMARY, "music_next");

    // Vol Up
    p.draw_button_with_icon(pw - 50, ctrl_y + 4, 34, 34, "media_vol_up", M3_SURFACE_CONTAINER_HIGH, M3_ON_SURFACE_VARIANT, "music_vol_up");

    // 7. Playlist Drawer (M3 Container Cards with Live Playing Indicator)
    let list_y = ctrl_y + 54;
    let item_w = p.width as u16 - 24;
    for (i, trk) in MUSIC_TRACKS.iter().enumerate() {
        let iy = list_y + (i as i16 * 36);
        if iy > (p.height as i16 - 60) { break; }
        let is_active = i == (state.music_track_idx % MUSIC_TRACKS.len());
        let bg = if is_active { M3_SURFACE_CONTAINER_HIGHEST } else { M3_SURFACE_CONTAINER_LOW };
        p.fill_rounded_rect(12, iy, item_w, 32, 10, bg);
        p.draw_rounded_outline(12, iy, item_w, 32, 10, if is_active { M3_PRIMARY } else { M3_OUTLINE_VARIANT });

        if is_active && state.music_playing {
            // Live 3-bar animated mini visualizer
            let v1 = ((state.music_tick as f32 * 0.2).sin().abs() * 10.0) as u16 + 3;
            let v2 = (((state.music_tick as f32 + 5.0) * 0.25).sin().abs() * 12.0) as u16 + 3;
            let v3 = (((state.music_tick as f32 + 10.0) * 0.18).sin().abs() * 9.0) as u16 + 3;
            p.fill_rect(20, iy + 22 - v1 as i16, 2, v1, M3_PRIMARY);
            p.fill_rect(24, iy + 22 - v2 as i16, 2, v2, M3_PRIMARY);
            p.fill_rect(28, iy + 22 - v3 as i16, 2, v3, M3_PRIMARY);
        } else {
            let num_str = format!("{}.", to_bengali_digits(&(i + 1).to_string()));
            p.draw_text_smooth(20, iy + 8, 11.5, &num_str, if is_active { M3_PRIMARY } else { M3_ON_SURFACE_VARIANT }, false);
        }

        p.draw_text_smooth(38, iy + 8, 11.5, trk.title, if is_active { M3_ON_SURFACE } else { M3_ON_SURFACE_VARIANT }, false);
        p.draw_text_smooth(pw - 52, iy + 8, 10.5, trk.duration, M3_ON_SURFACE_VARIANT, true);

        p.register_button(12, iy, item_w, 32, &format!("music_select_{}", i));
    }
}

fn render_control_center(p: &mut FramePainter, state: &SimState) {
    p.fill_rect(0, 0, p.width as u16, p.height as u16, M3_SURFACE);
    p.draw_text_smooth(16, 42, 19.0, "কন্ট্রোল সেন্টার (Control Center)", M3_PRIMARY, false);

    let card_w = ((p.width - 44) / 2) as u16;

    let toggles = [
        ("ওয়াই-ফাই", state.wifi_enabled, "toggle_wifi"),
        ("ব্লুটুথ", state.bt_enabled, "toggle_bt"),
        ("সফটবাস", state.softbus_enabled, "toggle_softbus"),
        ("ফ্ল্যাশলাইট", state.torch_enabled, "toggle_torch"),
        ("ডার্ক মোড", state.dark_mode, "toggle_theme"),
        ("মোবাইল ৫G", state.cellular_enabled, "toggle_cellular"),
        ("হটস্পট", state.hotspot_enabled, "toggle_hotspot"),
        ("ফ্লাইট মোড", state.airplane_mode, "toggle_airplane"),
    ];

    let start_y = 78;
    for (i, (label, enabled, id)) in toggles.iter().enumerate() {
        let col = (i % 2) as i16;
        let row = (i / 2) as i16;
        let x = 16 + col * (card_w as i16 + 12);
        let y = start_y + row * 64;

        let bg = if *enabled { M3_PRIMARY_CONTAINER } else { M3_SURFACE_CONTAINER_HIGH };
        let fg = if *enabled { M3_ON_PRIMARY_CONTAINER } else { M3_ON_SURFACE };
        let border = if *enabled { M3_PRIMARY } else { M3_OUTLINE_VARIANT };

        p.fill_rounded_rect(x, y, card_w, 56, 18, bg);
        p.draw_rounded_outline(x, y, card_w, 56, 18, border);

        p.draw_text_smooth(x + 12, y + 10, 14.0, label, fg, false);
        let stat = if *enabled { "[ সক্রিয় ]" } else { "[ বন্ধ ]" };
        let sub_fg = if *enabled { M3_ON_PRIMARY_CONTAINER } else { M3_ON_SURFACE_VARIANT };
        p.draw_text_smooth(x + 12, y + 32, 11.5, stat, sub_fg, false);

        p.register_button(x, y, card_w, 56, id);
    }

    let slider_y = start_y + 4 * 64 + 6;
    let sw = (p.width - 32) as u16;

    // M3 Brightness Slider Card
    p.fill_rounded_rect(16, slider_y, sw, 52, 16, M3_SURFACE_CONTAINER_LOW);
    p.draw_rounded_outline(16, slider_y, sw, 52, 16, M3_OUTLINE_VARIANT);
    p.draw_text_smooth(26, slider_y + 10, 13.5, &format!("ডিসপ্লে ব্রাইটনেস: {}%", to_bengali_digits(&state.brightness_level.to_string())), M3_ON_SURFACE, false);
    let br_filled = (((sw - 20) as f32 * (state.brightness_level as f32 / 100.0)) as u16).max(6).min(sw - 20);
    p.fill_pill(26, slider_y + 32, sw - 20, 8, M3_SURFACE_CONTAINER_HIGHEST);
    p.fill_pill(26, slider_y + 32, br_filled, 8, M3_PRIMARY);
    p.register_button(16, slider_y, sw, 52, "slider_brightness");

    // M3 Volume Slider Card
    p.fill_rounded_rect(16, slider_y + 60, sw, 52, 16, M3_SURFACE_CONTAINER_LOW);
    p.draw_rounded_outline(16, slider_y + 60, sw, 52, 16, M3_OUTLINE_VARIANT);
    p.draw_text_smooth(26, slider_y + 70, 13.5, &format!("মিডিয়া ভলিউম: {}%", to_bengali_digits(&state.volume_level.to_string())), M3_ON_SURFACE, false);
    let vol_filled = (((sw - 20) as f32 * (state.volume_level as f32 / 100.0)) as u16).max(6).min(sw - 20);
    p.fill_pill(26, slider_y + 92, sw - 20, 8, M3_SURFACE_CONTAINER_HIGHEST);
    p.fill_pill(26, slider_y + 92, vol_filled, 8, M3_GREEN);
    p.register_button(16, slider_y + 60, sw, 52, "slider_volume");

    p.draw_button(16, (p.height as i16) - 78, sw, 42, "বন্ধ করুন (Close Control Center)", M3_PRIMARY_CONTAINER, M3_ON_PRIMARY_CONTAINER, "close_control");
}

fn render_nano_editor(p: &mut FramePainter, state: &SimState) {
    p.fill_rect(0, 0, p.width as u16, p.height as u16, M3_SURFACE);
    
    // M3 App Bar
    p.fill_rect(0, 32, p.width as u16, 40, M3_SURFACE_CONTAINER);
    let title = format!("GNU nano 7.2 | {}{}", state.nano_filename, if state.nano_dirty { " [পরিবর্তিত]" } else { "" });
    p.draw_text_smooth(16, 43, 15.0, &title, M3_PRIMARY, false);

    let box_y = 78i16;
    let box_h = ((p.height as i16) - 100 - box_y).max(180) as u16;
    let box_w = (p.width - 24) as u16;
    let box_x = 12i16;

    // Terminal Container
    p.fill_rounded_rect(box_x, box_y, box_w, box_h, 14, M3_SURFACE_CONTAINER_LOWEST);
    p.draw_rounded_outline(box_x, box_y, box_w, box_h, 14, M3_OUTLINE_VARIANT);

    let line_height = 18;
    let max_lines = (box_h as usize / line_height).saturating_sub(1);

    for (i, line) in state.nano_lines.iter().enumerate().take(max_lines) {
        let ly = box_y + 8 + (i as i16 * line_height as i16);
        let num_str = format!("{:>2} | ", i + 1);
        p.draw_text_smooth(box_x + 8, ly, 11.5, &num_str, M3_ON_SURFACE_VARIANT, true);

        let line_x = box_x + 38;
        if i == state.nano_row {
            let chars: Vec<char> = line.chars().collect();
            let col = state.nano_col.min(chars.len());
            let before: String = chars[..col].iter().collect();
            let after: String = chars[col..].iter().collect();
            let cursor_char = if (state.term_cursor_ticks / 25) % 2 == 0 { "█" } else { " " };
            let render_line = format!("{}{}{}", before, cursor_char, after);
            p.draw_text_smooth(line_x, ly, 12.5, &render_line, M3_ON_SURFACE, true);
        } else {
            p.draw_text_smooth(line_x, ly, 12.5, line, M3_ON_SURFACE, true);
        }
    }

    let stat_y = box_y + box_h as i16 + 6;
    p.fill_pill(box_x, stat_y, box_w, 24, M3_SURFACE_CONTAINER_HIGH);
    p.draw_rounded_outline(box_x, stat_y, box_w, 24, 12, M3_OUTLINE_VARIANT);
    p.draw_text_smooth(box_x + 12, stat_y + 5, 11.5, &state.nano_status, M3_TEAL, false);

    let act_y = stat_y + 30;
    let btn_w = ((p.width - 32) / 3) as u16;
    p.draw_button(12, act_y, btn_w - 4, 32, "^O সেভ", M3_PRIMARY_CONTAINER, M3_ON_PRIMARY_CONTAINER, "nano_act_save");
    p.draw_button(12 + btn_w as i16, act_y, btn_w - 4, 32, "↵ নতুন লাইন", M3_SURFACE_CONTAINER_HIGH, M3_ON_SURFACE, "nano_act_enter");
    p.draw_button(12 + (2 * btn_w) as i16, act_y, btn_w - 4, 32, "^X প্রস্থান", M3_ERROR_CONTAINER, M3_ON_ERROR_CONTAINER, "nano_act_exit");
}

fn render_task_switcher(p: &mut FramePainter, state: &SimState) {
    p.fill_rect(0, 0, p.width as u16, p.height as u16, M3_SURFACE);

    // M3 App Header
    let start_y = 38i16;
    p.draw_text_smooth(16, start_y + 4, 18.0, "টাস্ক ম্যানেজার", M3_ON_SURFACE, true);
    p.draw_text_smooth(16, start_y + 26, 11.0, "র‍্যাম: ৩.৪ GB / ৮ GB (সক্রিয়)", M3_TEAL, false);

    // Clear all button at top right
    let clear_w = 90u16;
    let clear_h = 28u16;
    let clear_x = (p.width as i16) - 16 - clear_w as i16;
    let clear_y = start_y + 2;
    p.fill_pill(clear_x, clear_y, clear_w, clear_h, M3_PRIMARY_CONTAINER);
    p.draw_text_smooth(clear_x + 10, clear_y + 6, 11.0, "সব বন্ধ ✕", M3_ON_PRIMARY_CONTAINER, true);
    p.register_button(clear_x, clear_y, clear_w, clear_h, "tasks_clear_all");

    // Task list definition
    let tasks = [
        ("music", "app_music", "অনুরণ মিউজিক ও DSP", "৪-ভয়েস সিন্থেসাইজার সক্রিয়", "১৮.৪ MB", M3_ORANGE),
        ("terminal", "app_terminal", "NilVM / GNU Bash টার্মিনাল", "joy@nilos: ~/home/joy", "১২.১ MB", M3_GREEN),
        ("browser", "app_browser", "NilZar ওয়েব ব্রাউজার", "google.com (সক্রিয় ট্যাব)", "৩৪.৮ MB", M3_BLUE),
        ("notes", "app_notes", "ArkTS স্মার্ট নোটবুক", "৩টি নোট সিঙ্ক করা হয়েছে", "৮.৬ MB", M3_AMBER),
        ("messages", "app_messages", "মেসেজ ও SoftBus চ্যাট", "শেষ বার্তা: ১০:৩০ AM", "৯.২ MB", M3_TEAL),
        ("calc", "app_calc", "সায়েন্টিফিক ক্যালকুলেটর", "রিয়েলটাইম স্ট্যাক রেডি", "৬.৫ MB", M3_PRIMARY),
        ("files", "app_files", "NilFiles ফাইল এক্সপ্লোরার", "/home/joy (স্টোরেজ ভলিউম)", "১৪.২ MB", M3_BLUE),
        ("settings", "app_settings", "সিস্টেম সেটিংস", "NilHAL ভার্চুয়ালাইজার ওকে", "১১.০ MB", M3_PRIMARY),
    ];

    let card_w = (p.width as u16) - 24;
    let card_h = 92u16;
    let base_y = 78i16;

    for (idx, (id, icon, title, subtitle, ram, accent)) in tasks.iter().enumerate().take(5) {
        let cy = base_y + (idx as i16 * (card_h as i16 + 10));
        if cy + card_h as i16 > (p.height as i16 - 60) {
            break;
        }

        // Card Container
        p.fill_rounded_rect(12, cy, card_w, card_h, 12, M3_SURFACE_CONTAINER_HIGH);
        p.draw_rounded_outline(12, cy, card_w, card_h, 12, M3_OUTLINE_VARIANT);

        // Header inside card
        draw_vector_icon(p, 28, cy + 22, icon, *accent, 16);
        p.draw_text_smooth(46, cy + 14, 13.5, title, M3_ON_SURFACE, true);

        // RAM Pill
        let ram_w = 64u16;
        let ram_x = (p.width as i16) - 12 - 32 - ram_w as i16;
        p.fill_pill(ram_x, cy + 10, ram_w, 20, M3_SURFACE_CONTAINER_HIGHEST);
        p.draw_text_smooth(ram_x + 8, cy + 13, 10.0, ram, M3_ON_SURFACE_VARIANT, false);

        // Close '✕' Button
        let close_x = (p.width as i16) - 12 - 28;
        p.fill_circle_aa(close_x + 10, cy + 20, 9.0, M3_SURFACE_CONTAINER_HIGHEST);
        p.draw_text_smooth(close_x + 6, cy + 13, 11.0, "✕", M3_ON_SURFACE_VARIANT, true);
        p.register_button(close_x, cy + 10, 20, 20, &format!("task_close_{}", id));

        // Subtitle / Status
        p.draw_text_smooth(24, cy + 44, 11.5, subtitle, M3_ON_SURFACE_VARIANT, false);

        // Tap to open pill bar at bottom of card
        let open_btn_y = cy + 64;
        let open_btn_w = card_w - 24;
        p.fill_pill(24, open_btn_y, open_btn_w, 20, M3_SURFACE_CONTAINER_LOW);
        p.draw_text_smooth(36, open_btn_y + 3, 10.5, "➔ অ্যাপে ফিরে যান (ট্যাপ করুন)", M3_PRIMARY, false);

        // Register full card click to open
        p.register_button(12, cy, card_w, card_h, &format!("task_open_{}", id));
    }
}

fn render_app_terminal(p: &mut FramePainter, state: &SimState) {
    p.fill_rect(0, 0, p.width as u16, p.height as u16, M3_SURFACE);

    // M3 App Bar
    let mode_label = if state.python_mode { "Python 3.12" } else { "Onuron Bash 5.2" };
    p.fill_rect(0, 32, p.width as u16, 40, M3_SURFACE_CONTAINER);
    p.draw_text_smooth(16, 43, 16.0, &format!(">_ টার্মিনাল ({})", mode_label), M3_GREEN, false);
    p.draw_button((p.width as i16) - 76, 38, 64, 28, if state.python_mode { "bash" } else { "python" }, M3_SECONDARY_CONTAINER, M3_ON_SECONDARY_CONTAINER, "term_toggle_python");

    // Command Assist Chips
    let chip_w = ((p.width - 48) / 5) as u16;
    let chip_y = 78;
    let chips = if state.python_mode {
        ["print()", "2**10", "import os", "os.getcwd()", "exit()"]
    } else {
        ["ls -la", "pwd", "neofetch", "free -h", "clear"]
    };
    for (i, &cmd) in chips.iter().enumerate() {
        let cx = 12 + i as i16 * (chip_w as i16 + 4);
        p.draw_button(cx, chip_y, chip_w, 24, cmd, M3_SURFACE_CONTAINER_HIGH, M3_TEAL, &format!("term_chip_{}", cmd));
    }

    let box_y: i16 = 108;
    let box_h = ((p.height as i16) - 52 - 36 - box_y).max(200) as u16;
    let box_w = (p.width - 24) as u16;
    let box_x: i16 = 12;
    p.fill_rounded_rect(box_x, box_y, box_w, box_h, 14, M3_SURFACE_CONTAINER_LOWEST);
    p.draw_rounded_outline(box_x, box_y, box_w, box_h, 14, M3_OUTLINE_VARIANT);

    let line_height = 17usize;
    let max_visible_lines = (box_h as usize / line_height).saturating_sub(1);
    let mut all_display_lines = state.term_lines.clone();
    let cursor_char = if (state.term_cursor_ticks / 25) % 2 == 0 { "█" } else { " " };
    let (before_cursor, after_cursor) = state.term_cursor_split();
    let prompt_prefix = if state.python_mode {
        format!("py>>> {}{}{}", before_cursor, cursor_char, after_cursor)
    } else {
        format!("joy@onuron:{}$ {}{}{}", state.short_cwd(), before_cursor, cursor_char, after_cursor)
    };
    all_display_lines.push(TermLine { text: prompt_prefix, color: if state.python_mode { M3_AMBER } else { M3_ON_SURFACE } });

    let total_lines = all_display_lines.len();
    let max_scroll = total_lines.saturating_sub(max_visible_lines);
    let clamped_scroll = state.term_scroll_offset.min(max_scroll);
    let end_idx = total_lines.saturating_sub(clamped_scroll);
    let start_idx = end_idx.saturating_sub(max_visible_lines);

    for (i, line) in all_display_lines[start_idx..end_idx].iter().enumerate() {
        let ly = box_y + 8 + (i as i16 * line_height as i16);
        p.draw_text_smooth(box_x + 10, ly, 12.0, &line.text, line.color, true);
    }
    if total_lines > max_visible_lines {
        let bar_h = box_h.saturating_sub(16);
        let thumb_h = ((max_visible_lines as f32 / total_lines as f32) * bar_h as f32).clamp(16.0, bar_h as f32) as u16;
        let scroll_ratio = if max_scroll > 0 { (max_scroll - clamped_scroll) as f32 / max_scroll as f32 } else { 1.0 };
        let thumb_y = box_y + 8 + (scroll_ratio * (bar_h - thumb_h) as f32) as i16;
        p.fill_pill(box_x + box_w as i16 - 6, thumb_y, 4, thumb_h, M3_PRIMARY);
    }
}

fn render_app_phone(p: &mut FramePainter, state: &SimState) {
    let pw = p.width as i16;
    let cx = pw / 2;

    // M3 Top App Bar
    p.fill_rect(0, 32, p.width as u16, 40, M3_SURFACE_CONTAINER);
    p.draw_text_smooth(16, 43, 17.0, "ফোন ও ডায়ালার (VoLTE HD)", M3_PRIMARY, false);

    // Active in-call screen
    if state.phone_calling {
        let call_y: i16 = 96;
        // Avatar circle
        p.fill_rounded_rect(cx - 42, call_y, 84, 84, 42, M3_SURFACE_CONTAINER_HIGH);
        p.draw_rounded_outline(cx - 42, call_y, 84, 84, 42, M3_GREEN);
        draw_vector_icon(p, cx, call_y + 42, "phone", M3_GREEN, 32);

        let disp_name = if state.dial_number.is_empty() { "জয় সরকার" } else { &state.dial_number };
        let nw = p.text_width(20.0, disp_name, false);
        p.draw_text_smooth(cx - nw / 2, call_y + 98, 20.0, disp_name, M3_ON_SURFACE, false);

        p.draw_text_smooth(cx - 56, call_y + 126, 13.0, "VoLTE HD কল চলমান", M3_GREEN, false);

        // Call Timer
        let dur_m = state.phone_call_duration / 60;
        let dur_s = state.phone_call_duration % 60;
        let dur_str = to_bengali_digits(&format!("{:02}:{:02}", dur_m, dur_s));
        let tw = p.text_width(24.0, &dur_str, true);
        p.draw_text_smooth(cx - tw / 2, call_y + 152, 24.0, &dur_str, M3_ON_SURFACE, true);

        // Call actions: Mute, Keypad, Speaker
        let act_y = call_y + 208;
        let bw = (pw - 48) / 3;
        p.draw_button(16, act_y, bw as u16, 42, "মিউট", M3_SURFACE_CONTAINER_HIGH, M3_ON_SURFACE, "phone_mute");
        p.draw_button(16 + bw + 8, act_y, bw as u16, 42, "কীপ্যাড", M3_SURFACE_CONTAINER_HIGH, M3_ON_SURFACE, "phone_keypad");
        p.draw_button(16 + (bw + 8) * 2, act_y, bw as u16, 42, "স্পিকার", M3_SURFACE_CONTAINER_HIGH, M3_ON_SURFACE, "phone_speaker");

        // End Call big red FAB
        let end_y = act_y + 64;
        p.fill_rounded_rect(cx - 36, end_y, 72, 72, 36, M3_ERROR_CONTAINER);
        p.draw_rounded_outline(cx - 36, end_y, 72, 72, 36, M3_ERROR);
        let ew = p.text_width(16.0, "কাটুন", false);
        p.draw_text_smooth(cx - ew / 2, end_y + 26, 16.0, "কাটুন", M3_ON_ERROR_CONTAINER, false);
        p.register_button(cx - 36, end_y, 72, 72, "phone_end_call");
        return;
    }

    // Top M3 Segmented Buttons: ডায়ালার (0), সাম্প্রতিক (1), পরিচিতি (2)
    let tab_w = (p.width as u16 - 24) / 3;
    let tabs = [("ডায়ালার", 0), ("সাম্প্রতিক", 1), ("পরিচিতি", 2)];
    for (label, idx) in tabs {
        let tx = 12 + idx as i16 * (tab_w as i16 + 2);
        let is_sel = state.phone_tab == idx;
        let bg = if is_sel { M3_PRIMARY_CONTAINER } else { M3_SURFACE_CONTAINER_LOW };
        let fg = if is_sel { M3_ON_PRIMARY_CONTAINER } else { M3_ON_SURFACE_VARIANT };
        p.draw_button(tx, 76, tab_w, 32, label, bg, fg, &format!("phone_tab_{}", idx));
    }

    if state.phone_tab == 0 {
        // Tab 0: Dialer
        let num_w = (p.width - 24) as u16;
        p.fill_rounded_rect(12, 114, num_w, 46, 16, M3_SURFACE_CONTAINER_HIGH);
        p.draw_rounded_outline(12, 114, num_w, 46, 16, M3_OUTLINE_VARIANT);

        let (display_num, fg) = if state.dial_number.is_empty() {
            ("নম্বর লিখুন...", M3_ON_SURFACE_VARIANT)
        } else {
            (&state.dial_number as &str, M3_ON_SURFACE)
        };
        p.draw_text_smooth(24, 128, 19.0, display_num, fg, false);
        if !state.dial_number.is_empty() {
            p.draw_button(pw - 52, 119, 36, 36, "⌫", M3_SURFACE_CONTAINER_HIGHEST, M3_AMBER, "phone_del");
        }

        // Dialpad — 3x4 grid with M3 circular keys
        let pad_x = (pw - 240) / 2;
        let pad_y: i16 = 168;
        let btn_size: i16 = 68;
        let gap: i16 = 10;

        let keys = [
            [("1", ""), ("2", "ABC"), ("3", "DEF")],
            [("4", "GHI"), ("5", "JKL"), ("6", "MNO")],
            [("7", "PQRS"), ("8", "TUV"), ("9", "WXYZ")],
            [("*", ""), ("0", "+"), ("#", "")],
        ];

        for (r, row) in keys.iter().enumerate() {
            for (c, &(digit, sub)) in row.iter().enumerate() {
                let bx = pad_x + c as i16 * (btn_size + gap);
                let by = pad_y + r as i16 * (btn_size + gap);
                let id = format!("dial_key_{}", digit);

                p.fill_rounded_rect(bx, by, btn_size as u16, btn_size as u16, (btn_size / 2) as i16, M3_SURFACE_CONTAINER_HIGH);
                p.draw_rounded_outline(bx, by, btn_size as u16, btn_size as u16, (btn_size / 2) as i16, M3_OUTLINE_VARIANT);

                let dw = p.text_width(18.0, digit, false);
                let text_y = if sub.is_empty() { by + 24 } else { by + 16 };
                p.draw_text_smooth(bx + (btn_size - dw) / 2, text_y, 18.0, digit, M3_ON_SURFACE, false);
                if !sub.is_empty() {
                    let sw = p.text_width(9.0, sub, false);
                    p.draw_text_smooth(bx + (btn_size - sw) / 2, by + 40, 9.0, sub, M3_ON_SURFACE_VARIANT, false);
                }

                p.register_button(bx, by, btn_size as u16, btn_size as u16, &id);
            }
        }

        let act_y = pad_y + 4 * (btn_size + gap) + 4;
        p.fill_pill(pad_x, act_y, (btn_size * 3 + gap * 2) as u16, 46, M3_PRIMARY_CONTAINER);
        p.draw_rounded_outline(pad_x, act_y, (btn_size * 3 + gap * 2) as u16, 46, 23, M3_PRIMARY);
        let cw = p.text_width(16.0, "📞 কল করুন", false);
        p.draw_text_smooth(cx - cw / 2, act_y + 13, 16.0, "📞 কল করুন", M3_ON_PRIMARY_CONTAINER, false);
        p.register_button(pad_x, act_y, (btn_size * 3 + gap * 2) as u16, 46, "phone_call");
    } else if state.phone_tab == 1 {
        // Tab 1: Recent Calls List
        let list_w = (p.width - 24) as u16;
        let mut ry = 116i16;

        for (i, (name, meta, time, is_missed)) in state.phone_recents.iter().enumerate() {
            p.fill_rounded_rect(12, ry, list_w, 66, 16, M3_SURFACE_CONTAINER_LOW);
            p.draw_rounded_outline(12, ry, list_w, 66, 16, M3_OUTLINE_VARIANT);

            let tag_col = if *is_missed { M3_ERROR } else { M3_GREEN };
            p.fill_rounded_rect(12, ry, 4, 66, 2, tag_col);

            // Caller Name & Time
            p.draw_text_smooth(24, ry + 12, 14.5, name, M3_ON_SURFACE, false);
            p.draw_text_smooth(pw - 88, ry + 12, 11.5, time, M3_ON_SURFACE_VARIANT, false);

            // Call Type snippet
            p.draw_text_smooth(24, ry + 36, 12.0, meta, tag_col, false);

            // Call Button
            p.draw_button(pw - 66, ry + 16, 46, 34, "📞", M3_PRIMARY_CONTAINER, M3_ON_PRIMARY_CONTAINER, &format!("phone_call_recent_{}", i));

            ry += 74;
            if ry > (p.height as i16 - 70) { break; }
        }
    } else {
        // Tab 2: Contacts List
        let list_w = (p.width - 24) as u16;
        let mut cy_item = 116i16;

        for (i, (name, num, role)) in state.phone_contacts.iter().enumerate() {
            p.fill_rounded_rect(12, cy_item, list_w, 68, 16, M3_SURFACE_CONTAINER_LOW);
            p.draw_rounded_outline(12, cy_item, list_w, 68, 16, M3_OUTLINE_VARIANT);

            // Avatar circle
            p.fill_rounded_rect(20, cy_item + 14, 40, 40, 20, M3_SURFACE_CONTAINER_HIGH);
            p.draw_rounded_outline(20, cy_item + 14, 40, 40, 20, M3_OUTLINE_VARIANT);
            let first_ch = name.chars().next().unwrap_or('👤').to_string();
            let fw = p.text_width(15.0, &first_ch, false);
            p.draw_text_smooth(20 + (40 - fw) / 2, cy_item + 24, 15.0, &first_ch, M3_PRIMARY, false);

            // Name & Number
            p.draw_text_smooth(68, cy_item + 12, 14.5, name, M3_ON_SURFACE, false);
            p.draw_text_smooth(68, cy_item + 32, 11.5, num, M3_ON_SURFACE_VARIANT, false);
            p.draw_text_smooth(68, cy_item + 48, 10.5, role, M3_AMBER, false);

            // Call & Message buttons
            p.draw_button(pw - 96, cy_item + 18, 36, 32, "📞", M3_PRIMARY_CONTAINER, M3_ON_PRIMARY_CONTAINER, &format!("phone_call_contact_{}", i));
            p.draw_button(pw - 52, cy_item + 18, 36, 32, "✉", M3_SURFACE_CONTAINER_HIGH, M3_TEAL, &format!("phone_msg_contact_{}", i));

            cy_item += 76;
            if cy_item > (p.height as i16 - 70) { break; }
        }
    }
}

fn render_app_messages(p: &mut FramePainter, state: &SimState) {
    let pw = p.width as i16;

    // Header
    p.fill_rect(0, 32, p.width as u16, 40, M3_SURFACE_CONTAINER);

    // Thread Conversation View
    if let Some(thread_idx) = state.sms_active_thread {
        if let Some(thread) = state.sms_threads.get(thread_idx) {
            p.draw_button(12, 38, 72, 28, "< ব্যাক", M3_SURFACE_CONTAINER_HIGH, M3_PRIMARY, "msg_thread_back");
            p.draw_text_smooth(92, 44, 15.5, &format!("👤 {}", thread.name), M3_ON_SURFACE, true);

            let card_w = (p.width - 24) as u16;
            let mut bubble_y: i16 = 82;

            // Render message bubbles (take last 6 messages that fit)
            let skip_count = thread.messages.len().saturating_sub(6);
            for msg in thread.messages.iter().skip(skip_count) {
                if bubble_y > (p.height as i16 - 150) { break; }
                if msg.is_me {
                    // Outgoing bubble (right-aligned, M3 Primary Container)
                    let bw = (card_w - 30).min(230);
                    let bx = pw - 12 - bw as i16;
                    p.fill_rounded_rect(bx, bubble_y, bw, 50, 16, M3_PRIMARY_CONTAINER);
                    p.draw_rounded_outline(bx, bubble_y, bw, 50, 16, M3_PRIMARY);
                    p.draw_text_smooth(bx + 12, bubble_y + 10, 12.0, &msg.text, M3_ON_PRIMARY_CONTAINER, false);
                    p.draw_text_smooth(bx + 12, bubble_y + 30, 9.5, &format!("{} • প্রেরিত ✓✓", msg.time), M3_ON_PRIMARY_CONTAINER, false);
                } else {
                    // Incoming bubble (left-aligned, M3 Surface Container High)
                    let bw = (card_w - 30).min(240);
                    let bx = 12;
                    p.fill_rounded_rect(bx, bubble_y, bw, 50, 16, M3_SURFACE_CONTAINER_HIGH);
                    p.draw_rounded_outline(bx, bubble_y, bw, 50, 16, M3_OUTLINE_VARIANT);
                    p.draw_text_smooth(bx + 12, bubble_y + 10, 12.0, &msg.text, M3_ON_SURFACE, false);
                    p.draw_text_smooth(bx + 12, bubble_y + 30, 9.5, &msg.time, M3_ON_SURFACE_VARIANT, false);
                }
                bubble_y += 56;
            }

            // Quick Reply Action Chips (M3 Assist Chips)
            let qr_y = (p.height as i16) - 138;
            let quick_replies = [
                ("ঠিক আছে 👍", "msg_quick_ok"),
                ("আমি আসছি 🚗", "msg_quick_coming"),
                ("পরে বলি 📞", "msg_quick_later"),
                ("ফাইল পেয়েছি 📁", "msg_quick_gotfile"),
            ];
            let q_w = ((p.width - 36) / 4) as u16;
            for (i, (txt, q_id)) in quick_replies.iter().enumerate() {
                let qx = 12 + (i as i16 * (q_w as i16 + 4));
                p.draw_button(qx, qr_y, q_w, 26, txt, M3_SECONDARY_CONTAINER, M3_ON_SECONDARY_CONTAINER, q_id);
            }

            // Reply input bar at bottom
            let reply_y = (p.height as i16) - 104;
            p.fill_pill(12, reply_y, card_w - 74, 38, M3_SURFACE_CONTAINER_HIGH);
            p.draw_rounded_outline(12, reply_y, card_w - 74, 38, 19, M3_OUTLINE_VARIANT);
            let disp_reply = if state.sms_reply_input.is_empty() { "একটি উত্তর লিখুন..." } else { &state.sms_reply_input };
            let reply_fg = if state.sms_reply_input.is_empty() { M3_ON_SURFACE_VARIANT } else { M3_ON_SURFACE };
            p.draw_text_smooth(22, reply_y + 10, 12.5, disp_reply, reply_fg, false);

            p.draw_button(pw - 72, reply_y, 60, 38, "পাঠান", M3_PRIMARY_CONTAINER, M3_ON_PRIMARY_CONTAINER, "msg_send");
            return;
        }
    }

    p.draw_text_smooth(16, 43, 17.0, "বার্তা ও মেসেজিং (Material Messages)", M3_AMBER, false);

    // Main SMS thread list
    let btn_w = (p.width - 24) as u16;
    p.draw_button(12, 80, btn_w, 38, "+ নতুন বার্তা তৈরি করুন", M3_PRIMARY_CONTAINER, M3_ON_PRIMARY_CONTAINER, "msg_new");

    let list_y = 128;
    for (i, thread) in state.sms_threads.iter().enumerate() {
        let y = list_y + (i as i16 * 76);
        p.fill_rounded_rect(12, y, btn_w, 68, 16, M3_SURFACE_CONTAINER_LOW);
        p.draw_rounded_outline(12, y, btn_w, 68, 16, M3_OUTLINE_VARIANT);

        // Sender Avatar
        p.fill_rounded_rect(20, y + 14, 40, 40, 20, M3_SURFACE_CONTAINER_HIGH);
        p.draw_rounded_outline(20, y + 14, 40, 40, 20, M3_OUTLINE_VARIANT);
        let fw = p.text_width(16.0, &thread.avatar_char, false);
        p.draw_text_smooth(20 + (40 - fw) / 2, y + 24, 16.0, &thread.avatar_char, M3_PRIMARY, false);

        // Sender Name & Time
        let last_msg = thread.messages.last();
        let last_time = last_msg.map(|m| m.time.as_str()).unwrap_or("");
        let last_snippet = last_msg.map(|m| m.text.as_str()).unwrap_or("বার্তা নেই");

        p.draw_text_smooth(68, y + 14, 15.0, &thread.name, M3_ON_SURFACE, false);
        p.draw_text_smooth(pw - 64, y + 14, 11.5, last_time, M3_ON_SURFACE_VARIANT, false);

        // Preview snippet
        let snippet = safe_truncate(last_snippet, 28);
        p.draw_text_smooth(68, y + 38, 12.0, &snippet, M3_ON_SURFACE_VARIANT, false);

        let id = format!("msg_thread_{}", i);
        p.register_button(12, y, btn_w, 68, &id);
    }
}

fn render_app_files(p: &mut FramePainter, state: &SimState) {
    let pw = p.width as i16;

    // Header bar
    p.fill_rect(0, 32, p.width as u16, 40, M3_SURFACE_CONTAINER);
    p.draw_text_smooth(16, 43, 17.0, "ফাইল এক্সপ্লোরার (Material Files)", M3_BLUE, false);

    // Path breadcrumb formatted as clean virtual path in an M3 pill chip
    let virtual_curr = disk_to_virtual_display(&state.storage_root, &PathBuf::from(&state.current_path));
    let path_short = safe_truncate_end(&virtual_curr, 28);
    p.fill_pill(12, 78, (p.width - 24) as u16, 26, M3_SURFACE_CONTAINER_HIGH);
    p.draw_rounded_outline(12, 78, (p.width - 24) as u16, 26, 13, M3_OUTLINE_VARIANT);
    p.draw_text_smooth(22, 83, 11.5, &format!("পাথ: {}", path_short), M3_PRIMARY, false);

    // Quick-nav M3 chip row
    let bw = (p.width as u16 - 36) / 4;
    let navs = [("হোম", "bm_home"), ("উপরে", "bm_up"), ("রুট", "bm_root"), ("ডক্স", "bm_docs")];
    for (i, (lbl, id)) in navs.iter().enumerate() {
        let bx = 12 + i as i16 * (bw as i16 + 4);
        p.draw_button(bx, 110, bw, 26, lbl, M3_SECONDARY_CONTAINER, M3_ON_SECONDARY_CONTAINER, id);
    }

    // Action bar: + ফোল্ডার and + ফাইল
    let act_w = (p.width as u16 - 28) / 2;
    p.draw_button(12, 142, act_w, 30, "+ নতুন ফোল্ডার", M3_SURFACE_CONTAINER_HIGH, M3_GREEN, "file_new_folder");
    p.draw_button(12 + act_w as i16 + 4, 142, act_w, 30, "+ নতুন ফাইল", M3_SURFACE_CONTAINER_HIGH, M3_AMBER, "file_new_file");

    // File list
    let list_w = (p.width - 24) as u16;
    let mut list_y: i16 = 180;
    let max_visible: usize = ((p.height as i16 - 180 - 64) / 52).max(1) as usize;

    if let Ok(entries) = std::fs::read_dir(&state.current_path) {
        let mut items: Vec<(String, bool, u64)> = entries
            .flatten()
            .map(|e| {
                let name = e.file_name().to_string_lossy().to_string();
                let is_dir = e.file_type().map(|t| t.is_dir()).unwrap_or(false);
                let size = if !is_dir { e.metadata().map(|m| m.len()).unwrap_or(0) } else { 0 };
                (name, is_dir, size)
            })
            .collect();
        items.sort_by(|a, b| b.1.cmp(&a.1).then(a.0.cmp(&b.0)));

        if items.is_empty() {
            p.draw_text_smooth(24, list_y + 16, 14.0, "ফোল্ডারটি ফাঁকা", M3_ON_SURFACE_VARIANT, false);
        }

        for (name, is_dir, size) in items.iter().take(max_visible) {
            p.fill_rounded_rect(12, list_y, list_w, 46, 14, M3_SURFACE_CONTAINER_LOW);
            p.draw_rounded_outline(12, list_y, list_w, 46, 14, M3_OUTLINE_VARIANT);

            let (badge, icon_col) = if *is_dir {
                ("DIR", M3_AMBER)
            } else if name.ends_with(".mp4") || name.ends_with(".mkv") || name.ends_with(".avi") {
                ("VID", M3_ORANGE)
            } else if name.ends_with(".mp3") || name.ends_with(".ogg") {
                ("AUD", M3_TEAL)
            } else if name.ends_with(".rs") || name.ends_with(".c") || name.ends_with(".py") {
                ("SRC", M3_GREEN)
            } else if name.ends_with(".txt") || name.ends_with(".md") {
                ("DOC", M3_PRIMARY)
            } else {
                ("FILE", M3_ON_SURFACE_VARIANT)
            };
            p.fill_pill(18, list_y + 11, 30, 24, blend_pixel(M3_SURFACE_CONTAINER_HIGH, icon_col, 50));
            let bw = p.text_width(9.5, badge, true);
            p.draw_text_smooth(18 + (30 - bw) / 2, list_y + 17, 9.5, badge, icon_col, true);

            let display_name = safe_truncate(name, 16);
            p.draw_text_smooth(54, list_y + 8, 13.5, &display_name, if *is_dir { M3_PRIMARY } else { M3_ON_SURFACE }, false);

            let meta = if *is_dir {
                "ফোল্ডার".to_string()
            } else if *size > 1_000_000 {
                format!("{:.1} MB", *size as f64 / 1_000_000.0)
            } else if *size > 1_000 {
                format!("{} KB", size / 1_000)
            } else {
                format!("{} B", size)
            };
            p.draw_text_smooth(54, list_y + 27, 10.5, &meta, M3_ON_SURFACE_VARIANT, false);

            let id = format!("file_open_{}", name);
            p.register_button(12, list_y, list_w - 68, 46, &id);
            if !is_dir {
                p.draw_button(pw - 64, list_y + 9, 48, 28, "খোল", M3_PRIMARY_CONTAINER, M3_ON_PRIMARY_CONTAINER, &format!("file_act_{}", name));
            }
            list_y += 52;
        }
    } else {
        p.draw_text_smooth(24, list_y + 16, 14.0, &format!("ডিরেক্টরি পড়া যায়নি: {}", state.current_path), M3_ERROR, false);
    }
}

fn render_app_settings(p: &mut FramePainter, state: &SimState) {
    let pw = p.width as i16;

    // Header
    p.fill_rect(0, 32, p.width as u16, 40, M3_SURFACE_CONTAINER);
    p.draw_text_smooth(16, 43, 17.0, "সেটিংস ও ডিভাইস কনফিগ", M3_SECONDARY, false);

    let card_w = (p.width - 24) as u16;
    let mut y = 78i16;

    // Settings Items in M3 Container Cards
    let settings_items = [
        ("ওয়াই-ফাই নেটওয়ার্ক", state.wifi_enabled, "toggle_wifi", "Wi-Fi 7 (802.11be)"),
        ("মোবাইল নেটওয়ার্ক ও ৫G", state.cellular_enabled, "toggle_cellular", "Jio True 5G SA"),
        ("ব্লুটুথ সংযোগ", state.bt_enabled, "toggle_bt", "Bluetooth 5.4 LE Audio"),
        ("সফটবাস ডিভাইস মেশ", state.softbus_enabled, "toggle_softbus", "P2P QUIC Fabric"),
    ];

    for (title, enabled, id, sub) in settings_items {
        p.fill_rounded_rect(12, y, card_w, 56, 16, M3_SURFACE_CONTAINER_LOW);
        p.draw_rounded_outline(12, y, card_w, 56, 16, M3_OUTLINE_VARIANT);

        p.draw_text_smooth(24, y + 10, 14.5, title, M3_ON_SURFACE, false);
        p.draw_text_smooth(24, y + 32, 11.5, sub, M3_ON_SURFACE_VARIANT, false);

        let (stat_lbl, bg, fg) = if enabled {
            ("চালু", M3_PRIMARY_CONTAINER, M3_ON_PRIMARY_CONTAINER)
        } else {
            ("বন্ধ", M3_SURFACE_CONTAINER_HIGH, M3_ON_SURFACE_VARIANT)
        };
        p.fill_pill(pw - 66, y + 14, 52, 28, bg);
        p.draw_rounded_outline(pw - 66, y + 14, 52, 28, 14, if enabled { M3_PRIMARY } else { M3_OUTLINE_VARIANT });
        let tw = p.text_width(12.0, stat_lbl, false);
        p.draw_text_smooth(pw - 66 + (52 - tw) / 2, y + 21, 12.0, stat_lbl, fg, false);

        p.register_button(12, y, card_w, 56, id);
        y += 64;
    }

    // Signal Strength Item
    p.fill_rounded_rect(12, y, card_w, 56, 16, M3_SURFACE_CONTAINER_LOW);
    p.draw_rounded_outline(12, y, card_w, 56, 16, M3_OUTLINE_VARIANT);
    p.draw_text_smooth(24, y + 10, 14.5, "নেটওয়ার্ক টাওয়ার শক্তি", M3_ON_SURFACE, false);
    let sig_text = if state.cellular_enabled {
        format!("{} বার সক্রিয়", to_bengali_digits(&state.cellular_signal_level.to_string()))
    } else {
        "কোনো সিগন্যাল নেই".to_string()
    };
    p.draw_text_smooth(24, y + 32, 11.5, &sig_text, M3_TEAL, false);
    p.draw_button(pw - 78, y + 14, 64, 28, "পরিবর্তন", M3_SECONDARY_CONTAINER, M3_ON_SECONDARY_CONTAINER, "cycle_signal");
    p.register_button(12, y, card_w - 70, 56, "cycle_signal");
    y += 64;

    // System Info Card (M3 Elevated)
    p.fill_rounded_rect(12, y, card_w, 112, 16, M3_SURFACE_CONTAINER_HIGH);
    p.draw_rounded_outline(12, y, card_w, 112, 16, M3_PRIMARY);
    p.draw_text_smooth(24, y + 12, 15.0, "অনুরণ ওএস সিস্টেম তথ্য", M3_PRIMARY, false);
    p.draw_text_smooth(24, y + 36, 12.5, "ভার্সন: Onuron OS 1.7 (Material Design 3)", M3_ON_SURFACE, false);
    p.draw_text_smooth(24, y + 58, 12.0, "কার্নেল: Linux 6.6.21-nil-aarch64 (HAL v2)", M3_GREEN, false);
    p.draw_text_smooth(24, y + 80, 12.0, "টাইমজোন: Indian Standard Time (IST UTC+5:30)", M3_ON_SURFACE_VARIANT, false);
}

fn render_app_nilpkg(p: &mut FramePainter, state: &SimState) {
    let pw = p.width as i16;

    // Header
    p.fill_rect(0, 32, p.width as u16, 40, M3_SURFACE_CONTAINER);
    p.draw_text_smooth(16, 43, 17.0, "নীলপ্যাকেজ স্টোর (Material NilPkg)", M3_PINK, false);

    let card_w = (p.width - 24) as u16;
    let packages = [
        ("org.videolan.vlc", "ভিএলসি মিডিয়া প্লেয়ার (VLC VideoLAN)", M3_ORANGE),
        ("org.mozilla.fenix", "ফায়ারফক্স প্রাইভেট ব্রাউজার", M3_AMBER),
        ("com.signal.android", "সিগন্যাল এনক্রিপ্টেড চ্যাট", M3_BLUE),
        ("org.openstreetmap", "অর্গানিক অফলাইন ম্যাপ ও নেভিগেশন", M3_GREEN),
    ];

    let mut y = 80i16;
    for (id_name, desc, tag_color) in packages {
        p.fill_rounded_rect(12, y, card_w, 72, 16, M3_SURFACE_CONTAINER_LOW);
        p.draw_rounded_outline(12, y, card_w, 72, 16, M3_OUTLINE_VARIANT);
        p.fill_rounded_rect(12, y, 4, 72, 2, tag_color);

        let is_installed = state.is_pkg_installed(id_name);
        p.draw_text_smooth(24, y + 12, 14.5, id_name, M3_ON_SURFACE, false);
        p.draw_text_smooth(24, y + 36, 12.0, desc, M3_ON_SURFACE_VARIANT, false);

        let (lbl, bg, fg) = if is_installed {
            ("খুলুন", M3_SECONDARY_CONTAINER, M3_ON_SECONDARY_CONTAINER)
        } else {
            ("ইনস্টল", M3_PRIMARY_CONTAINER, M3_ON_PRIMARY_CONTAINER)
        };

        let btn_x = pw - 86;
        p.draw_button(btn_x, y + 20, 70, 32, lbl, bg, fg, &format!("pkg_act_{}", id_name));
        y += 80;
    }
}

fn render_app_softbus(p: &mut FramePainter, _state: &SimState) {
    let pw = p.width as i16;

    // Header
    p.fill_rect(0, 32, p.width as u16, 40, M3_SURFACE_CONTAINER);
    p.draw_text_smooth(16, 43, 17.0, "সফটবাস ডিভাইস মেশ (Material SoftBus)", M3_TEAL, false);

    p.draw_text_smooth(16, 78, 12.5, "ডিস্ট্রিবিউটেড মেশ নেটওয়ার্ক (QUIC Fabric)", M3_ON_SURFACE_VARIANT, false);

    let card_w = (p.width - 24) as u16;
    let devices = [
        ("NilPad-Pro-X1", "যুক্ত (২ মি.সে.) — ক্লিপবোর্ড সিঙ্ক সক্রিয়", M3_GREEN, "নিলপ্যাড"),
        ("NilBook-Ultra", "যুক্ত (QUIC) — স্ক্রিন কাস্টিং প্রস্তুত", M3_GREEN, "নিলবুক"),
        ("NilVision-65", "কাছে পাওয়া গেছে — ৪K ওয়্যারলেস রেডি", M3_AMBER, "টিভি"),
    ];

    let mut y = 100i16;
    for (name, caps, color, short) in devices {
        p.fill_rounded_rect(12, y, card_w, 72, 16, M3_SURFACE_CONTAINER_LOW);
        p.draw_rounded_outline(12, y, card_w, 72, 16, M3_OUTLINE_VARIANT);
        p.fill_rounded_rect(12, y, 4, 72, 2, color);

        p.draw_text_smooth(24, y + 12, 14.5, name, M3_ON_SURFACE, false);
        p.draw_text_smooth(24, y + 36, 12.0, caps, M3_ON_SURFACE_VARIANT, false);

        let btn_x = pw - 96;
        p.draw_button(btn_x, y + 18, 80, 34, "ফাইল ড্রপ", M3_PRIMARY_CONTAINER, M3_ON_PRIMARY_CONTAINER, &format!("sb_drop_{}", short));
        y += 80;
    }

    let act_y = y + 12;
    p.draw_button(12, act_y, card_w, 44, "নতুন ডিভাইস স্ক্যান করুন (Scan Mesh)", M3_SECONDARY_CONTAINER, M3_ON_SECONDARY_CONTAINER, "sb_scan");
}

fn render_app_android(p: &mut FramePainter, _state: &SimState) {
    // Header
    p.fill_rect(0, 32, p.width as u16, 40, M3_SURFACE_CONTAINER);
    p.draw_text_smooth(16, 43, 17.0, "অ্যান্ড্রয়েড কন্টেইনার (AOSP 14)", M3_GREEN, false);

    let card_w = (p.width - 24) as u16;
    p.fill_rounded_rect(12, 80, card_w, 180, 16, M3_SURFACE_CONTAINER_LOW);
    p.draw_rounded_outline(12, 80, card_w, 180, 16, M3_OUTLINE_VARIANT);

    p.draw_text_smooth(24, 80 + 16, 14.5, "ইঞ্জিন: LXC আইসোলেটেড কন্টেইনার", M3_ON_SURFACE, false);
    p.draw_text_smooth(24, 80 + 44, 14.0, "রানটাইম: AOSP 14 হেডলেস কোর", M3_GREEN, false);
    p.draw_text_smooth(24, 80 + 72, 13.5, "ব্রিজ: বাইন্ডার-শিম পাসথ্রু", M3_ON_SURFACE_VARIANT, false);
    p.draw_text_smooth(24, 80 + 100, 13.5, "মাইক্রোজি: ইউনিফায়েড পুশ সক্রিয়", M3_TEAL, false);
    p.draw_text_smooth(24, 80 + 128, 13.5, "হার্ডওয়্যার আইডি: মাস্কড (অ্যান্টি-ট্র্যাক)", M3_AMBER, false);

    p.draw_button(12, 276, card_w, 46, "রানটাইম পুনরায় চালু করুন", M3_PRIMARY_CONTAINER, M3_ON_PRIMARY_CONTAINER, "aosp_restart");
}

// ─── Input Callback Handler ───────────────────────────────────────────────────
struct InputHandler {
    typed_chars: Arc<Mutex<Vec<char>>>,
}

impl minifb::InputCallback for InputHandler {
    fn add_char(&mut self, uni_char: u32) {
        if let Some(ch) = char::from_u32(uni_char) {
            if is_valid_input_char(ch) {
                if let Ok(mut lock) = self.typed_chars.lock() {
                    lock.push(ch);
                }
            }
        }
    }
}

// ─── Main Desktop Simulator Loop ──────────────────────────────────────────────
fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("=========================================================");
    println!("  Onuron OS Mobile Simulator (Alap Framework & Chromium)  ");
    println!("=========================================================");
    println!("[*] Initializing Onuron Core Systems, Alap Engine, and NilZar Chromium Browser...");

    let fonts = FontEngine::load();
    let mut buffer: Vec<u32> = vec![COLOR_BG; SCREEN_WIDTH * SCREEN_HEIGHT];

    let mut window = Window::new(
        "Onuron Mobile Simulator (Chromium V8 & Alap Engine)",
        SCREEN_WIDTH,
        SCREEN_HEIGHT,
        WindowOptions {
            resize: false,
            scale: Scale::X1,
            ..Default::default()
        },
    )?;

    #[cfg(target_os = "windows")]
    let parent_hwnd = window.get_window_handle() as isize;
    #[cfg(target_os = "windows")]
    let mut embedded_browser = nilbrowser::chromium::create_embedded_browser(
        parent_hwnd,
        4, 126, 322, 500,
        "https://www.google.com",
    ).ok();

    #[allow(deprecated)]
    window.limit_update_rate(Some(std::time::Duration::from_micros(16600))); // 60 FPS

    let typed_chars = Arc::new(Mutex::new(Vec::new()));

    let handler = Box::new(InputHandler {
        typed_chars: Arc::clone(&typed_chars),
    });

    window.set_input_callback(handler);

    let mut state = SimState::new();
    let audio_engine = AudioEngine::new();
    audio_engine.set_playing(state.music_playing);
    audio_engine.set_track(state.music_track_idx);
    audio_engine.set_volume(state.volume_level);
    let mut was_mouse_down = false;

    println!("[OK] Simulator running at {}x{} with Onuron OS & Alap Engine.", SCREEN_WIDTH, SCREEN_HEIGHT);

    while window.is_open() && !window.is_key_down(Key::Escape) {
        state.term_cursor_ticks = state.term_cursor_ticks.wrapping_add(1);

        // Advance media progress tick
        if state.music_playing {
            state.music_tick = state.music_tick.wrapping_add(1);
            if state.music_tick % 60 == 0 {
                let dur = MUSIC_TRACKS[state.music_track_idx % MUSIC_TRACKS.len()].duration_secs;
                state.music_current_sec = (state.music_current_sec + 1) % dur;
                state.music_progress = state.music_current_sec as f32 / dur as f32;
            }
        }
        if state.phone_calling && state.term_cursor_ticks % 60 == 0 {
            state.phone_call_duration += 1;
        }

        // Live simulated SMS auto-reply countdown
        if let Some((ticks, thread_idx, reply_text)) = state.sms_auto_reply_ticks.take() {
            if ticks == 0 {
                if let Some(th) = state.sms_threads.get_mut(thread_idx) {
                    let s_name = th.name.clone();
                    th.messages.push(ChatMessageItem {
                        sender: s_name,
                        text: reply_text,
                        time: get_ist_time_str(),
                        is_me: false,
                    });
                    audio_engine.play_unlock_sound();
                }
            } else {
                state.sms_auto_reply_ticks = Some((ticks - 1, thread_idx, reply_text));
            }
        }

        // 1. Mouse Scroll Wheel
        if let Some((_x, y_scroll)) = window.get_scroll_wheel() {
            if state.screen == Screen::AppTerminal {
                if y_scroll > 0.0 {
                    state.term_scroll_offset = state.term_scroll_offset.saturating_add(3);
                } else if y_scroll < 0.0 {
                    state.term_scroll_offset = state.term_scroll_offset.saturating_sub(3);
                }
            } else if state.screen == Screen::AppBrowser {
                if y_scroll < 0.0 {
                    state.browser_scroll_offset = state.browser_scroll_offset.saturating_add(2);
                } else if y_scroll > 0.0 {
                    state.browser_scroll_offset = state.browser_scroll_offset.saturating_sub(2);
                }
            }
        }

        // 2. Process Physical Keyboard Characters
        if let Ok(mut chars) = typed_chars.lock() {
            for ch in chars.drain(..) {
                match state.screen {
                    Screen::AppTerminal => {
                        state.term_insert_char(ch);
                    }
                    Screen::AppBrowser => {
                        state.browser_url_insert_char(ch);
                    }
                    Screen::NanoEditor => {
                        state.nano_insert_char(ch);
                    }
                    Screen::AppNotes => {
                        if state.notes_editing {
                            state.note_insert_char(ch);
                        }
                    }
                    Screen::AppMessages => {
                        if ch != '\r' && ch != '\n' && is_valid_input_char(ch) {
                            state.sms_reply_input.push(ch);
                        }
                    }
                    Screen::AppCalculator => {
                        if ch.is_ascii_digit() || ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '.' || ch == '%' || (ch >= '০' && ch <= '৯') {
                            audio_engine.play_click();
                            let digit_str = match ch {
                                '*' => "×".to_string(),
                                '/' => "÷".to_string(),
                                other => other.to_string(),
                            };
                            state.exec_calc_press(&digit_str);
                        }
                    }
                    Screen::Lockscreen => {
                        if (ch.is_ascii_digit() || (ch >= '০' && ch <= '৯')) && state.pin_input.len() < 4 {
                            audio_engine.play_dtmf(ch);
                            let digit_char = match ch {
                                '০' => '0', '১' => '1', '২' => '2', '৩' => '3', '৪' => '4',
                                '৫' => '5', '৬' => '6', '৭' => '7', '৮' => '8', '৯' => '9',
                                _ => ch,
                            };
                            state.pin_input.push(digit_char);
                            if state.pin_input.len() == 4 {
                                if state.pin_input == state.pin {
                                    audio_engine.play_unlock_sound();
                                    state.screen = Screen::Home;
                                    state.pin_input.clear();
                                    state.lock_error = false;
                                } else {
                                    state.lock_error = true;
                                }
                            }
                        }
                    }
                    Screen::AppPhone => {
                        if ch.is_ascii_digit() || ch == '*' || ch == '#' || ch == '+' || (ch >= '০' && ch <= '৯') {
                            audio_engine.play_dtmf(ch);
                            state.dial_number.push(ch);
                        }
                    }
                    _ => {}
                }
            }
        }

        let ctrl_down = window.is_key_down(Key::LeftCtrl) || window.is_key_down(Key::RightCtrl);

        // 3. Process Special Keys
        for key in window.get_keys_pressed(KeyRepeat::Yes) {
            match state.screen {
                Screen::AppBrowser => match key {
                    Key::Enter => {
                        let query = state.browser_url_input.clone();
                        state.browser_fetch_live(&query);
                    }
                    Key::Backspace => {
                        state.browser_url_backspace();
                    }
                    Key::PageDown | Key::Down => {
                        state.browser_scroll_offset = state.browser_scroll_offset.saturating_add(3);
                    }
                    Key::PageUp | Key::Up => {
                        state.browser_scroll_offset = state.browser_scroll_offset.saturating_sub(3);
                    }
                    _ => {}
                },
                Screen::AppMessages => match key {
                    Key::Enter => {
                        if !state.sms_reply_input.is_empty() {
                            let text = std::mem::take(&mut state.sms_reply_input);
                            if let Some(t_idx) = state.sms_active_thread {
                                if let Some(th) = state.sms_threads.get_mut(t_idx) {
                                    th.messages.push(ChatMessageItem {
                                        sender: "আমি".into(),
                                        text: text.clone(),
                                        time: get_ist_time_str(),
                                        is_me: true,
                                    });
                                    audio_engine.play_msg_sent_sound();
                                    state.trigger_sms_reply(t_idx, &text);
                                }
                            }
                        }
                    }
                    Key::Backspace => {
                        state.sms_reply_input.pop();
                    }
                    _ => {}
                },
                Screen::NanoEditor => match key {
                    Key::O if ctrl_down => {
                        state.nano_save();
                    }
                    Key::S if ctrl_down => {
                        state.nano_save();
                    }
                    Key::X if ctrl_down => {
                        state.screen = Screen::AppTerminal;
                    }
                    Key::Enter => {
                        state.nano_enter();
                    }
                    Key::Backspace => {
                        state.nano_backspace();
                    }
                    Key::Left => {
                        state.nano_col = state.nano_col.saturating_sub(1);
                    }
                    Key::Right => {
                        if state.nano_row < state.nano_lines.len() {
                            let len = state.nano_lines[state.nano_row].chars().count();
                            if state.nano_col < len {
                                state.nano_col += 1;
                            }
                        }
                    }
                    Key::Up => {
                        state.nano_row = state.nano_row.saturating_sub(1);
                    }
                    Key::Down => {
                        if state.nano_row + 1 < state.nano_lines.len() {
                            state.nano_row += 1;
                        }
                    }
                    _ => {}
                },
                Screen::AppTerminal => match key {
                    Key::Left => {
                        state.term_cursor_pos = state.term_cursor_pos.saturating_sub(1);
                    }
                    Key::Right => {
                        let count = state.term_char_count();
                        if state.term_cursor_pos < count {
                            state.term_cursor_pos += 1;
                        }
                    }
                    Key::Home => {
                        state.term_cursor_pos = 0;
                    }
                    Key::End => {
                        state.term_cursor_pos = state.term_char_count();
                    }
                    Key::Delete => {
                        state.term_delete();
                    }
                    Key::Enter => {
                        let cmd = std::mem::take(&mut state.term_input);
                        state.term_cursor_pos = 0;
                        state.exec_term_command(&cmd);
                    }
                    Key::Backspace => {
                        state.term_backspace();
                    }
                    Key::Up => {
                        if !state.term_history.is_empty() {
                            let new_idx = match state.term_history_idx {
                                None => state.term_history.len().saturating_sub(1),
                                Some(i) => i.saturating_sub(1),
                            };
                            state.term_history_idx = Some(new_idx);
                            state.term_input = state.term_history[new_idx].clone();
                            state.term_cursor_pos = state.term_char_count();
                        }
                    }
                    Key::Down => {
                        if let Some(i) = state.term_history_idx {
                            if i + 1 < state.term_history.len() {
                                state.term_history_idx = Some(i + 1);
                                state.term_input = state.term_history[i + 1].clone();
                                state.term_cursor_pos = state.term_char_count();
                            } else {
                                state.term_history_idx = None;
                                state.term_input.clear();
                                state.term_cursor_pos = 0;
                            }
                        }
                    }
                    Key::PageUp => {
                        state.term_scroll_offset = state.term_scroll_offset.saturating_add(8);
                    }
                    Key::PageDown => {
                        state.term_scroll_offset = state.term_scroll_offset.saturating_sub(8);
                    }
                    _ => {}
                },
                Screen::AppNotes => {
                    if state.notes_editing {
                        match key {
                            Key::Backspace => {
                                state.note_backspace();
                            }
                            Key::Enter => {
                                state.note_insert_char('\n');
                            }
                            Key::Left => {
                                state.notes_cursor_pos = state.notes_cursor_pos.saturating_sub(1);
                            }
                            Key::Right => {
                                let len = state.notes_edit_content.chars().count();
                                if state.notes_cursor_pos < len {
                                    state.notes_cursor_pos += 1;
                                }
                            }
                            _ => {}
                        }
                    }
                }
                Screen::AppCalculator => match key {
                    Key::Enter => {
                        state.exec_calc_press("=");
                    }
                    Key::Backspace => {
                        if state.calc_expr.len() > 1 {
                            state.calc_expr.pop();
                        } else {
                            state.calc_expr = "০".into();
                        }
                    }
                    _ => {}
                },
                Screen::Lockscreen => match key {
                    Key::Enter => {
                        if state.pin_input == state.pin {
                            state.screen = Screen::Home;
                            state.pin_input.clear();
                            state.lock_error = false;
                        } else {
                            state.lock_error = true;
                        }
                    }
                    Key::Backspace => {
                        state.pin_input.pop();
                    }
                    _ => {}
                },
                _ => {}
            }
        }

        // 4. Render Frame to Buffer
        buffer.fill(COLOR_BG);

        let mut painter = FramePainter::new(&mut buffer, SCREEN_WIDTH, SCREEN_HEIGHT, &fonts);

        match &state.screen {
            Screen::Lockscreen => render_lockscreen(&mut painter, &state),
            Screen::Home => render_home(&mut painter, &state),
            Screen::AppPhone => render_app_phone(&mut painter, &state),
            Screen::AppMessages => render_app_messages(&mut painter, &state),
            Screen::AppFiles => render_app_files(&mut painter, &state),
            Screen::AppSettings => render_app_settings(&mut painter, &state),
            Screen::AppNilPkg => render_app_nilpkg(&mut painter, &state),
            Screen::AppSoftBus => render_app_softbus(&mut painter, &state),
            Screen::AppAndroid => render_app_android(&mut painter, &state),
            Screen::AppTerminal => render_app_terminal(&mut painter, &state),
            Screen::AppNotes => render_app_notes(&mut painter, &state),
            Screen::AppCalculator => render_app_calculator(&mut painter, &state),
            Screen::AppMusic => render_app_music(&mut painter, &state, &audio_engine.get_spectrum()),
            Screen::AppBrowser => render_app_browser(&mut painter, &state),
            Screen::ControlCenter => render_control_center(&mut painter, &state),
            Screen::NanoEditor => render_nano_editor(&mut painter, &state),
            Screen::TaskSwitcher => render_task_switcher(&mut painter, &state),
        }

        // Live Dynamic Status Bar
        if state.screen != Screen::ControlCenter && state.screen != Screen::NanoEditor {
            render_status_bar(&mut painter, &state);
        }

        if state.screen != Screen::Lockscreen && state.screen != Screen::NanoEditor && state.screen != Screen::ControlCenter {
            render_bottom_nav(&mut painter, &state.screen);
        }

        let buttons = painter.buttons.clone();

        // 5. Process Mouse Touch Interactions
        let mouse_down = window.get_mouse_down(MouseButton::Left);
        if mouse_down {
            #[cfg(target_os = "windows")]
            if let Some((_mx, my)) = window.get_mouse_pos(MouseMode::Pass) {
                if my < 126.0 || my >= 628.0 {
                    unsafe {
                        SetFocus(parent_hwnd);
                    }
                }
            }
        }
        if mouse_down && !was_mouse_down {
            if let Some((mx, my)) = window.get_mouse_pos(MouseMode::Pass) {
                let px = mx as i16;
                let py = my as i16;

                if let Some(btn) = buttons.iter().find(|b| b.contains(px, py)) {
                    let id = &btn.id;
                    audio_engine.play_click();

                    if id == "nav_back" || id == "nav_home" {
                        state.screen = Screen::Home;
                    } else if id == "nav_tasks" {
                        if state.screen == Screen::TaskSwitcher {
                            state.screen = Screen::Home;
                        } else {
                            state.screen = Screen::TaskSwitcher;
                        }
                    } else if id == "tasks_clear_all" {
                        state.push_term_line("[টাস্ক ম্যানেজার] সমস্ত ব্যাকগ্রাউন্ড প্রসেস পরিষ্কার করা হয়েছে।".into(), COLOR_GREEN);
                        state.screen = Screen::Home;
                    } else if id.starts_with("task_open_") {
                        let target = id.trim_start_matches("task_open_");
                        match target {
                            "music" => state.screen = Screen::AppMusic,
                            "term" | "terminal" => state.screen = Screen::AppTerminal,
                            "browser" => state.screen = Screen::AppBrowser,
                            "notes" => state.screen = Screen::AppNotes,
                            "messages" => state.screen = Screen::AppMessages,
                            "calc" | "calculator" => state.screen = Screen::AppCalculator,
                            "files" => state.screen = Screen::AppFiles,
                            "settings" => state.screen = Screen::AppSettings,
                            "nilpkg" => state.screen = Screen::AppNilPkg,
                            "softbus" => state.screen = Screen::AppSoftBus,
                            "phone" => state.screen = Screen::AppPhone,
                            _ => state.screen = Screen::Home,
                        }
                    } else if id.starts_with("task_close_") {
                        let target = id.trim_start_matches("task_close_");
                        state.push_term_line(format!("[টাস্ক] অ্যাপ বন্ধ করা হয়েছে: {}", target), COLOR_AMBER);
                    } else if id == "nav_lock" {
                        state.pin_input.clear();
                        state.lock_error = false;
                        state.screen = Screen::Lockscreen;
                        audio_engine.play_lock_sound();
                    } else if id == "toggle_island" {
                        state.screen = Screen::ControlCenter;
                    } else if id == "close_control" {
                        state.screen = Screen::Home;
                    } else if id == "lock_torch" {
                        state.torch_enabled = !state.torch_enabled;
                    } else if id == "lock_emergency" {
                        state.screen = Screen::AppPhone;
                        state.dial_number = "112".into();
                        state.phone_tab = 0;
                    } else if id == "app_phone" {
                        state.screen = Screen::AppPhone;
                    } else if id == "phone_tab_0" {
                        state.phone_tab = 0;
                    } else if id == "phone_tab_1" {
                        state.phone_tab = 1;
                    } else if id == "phone_tab_2" {
                        state.phone_tab = 2;
                    } else if id.starts_with("phone_call_recent_") {
                        if let Ok(idx) = id.trim_start_matches("phone_call_recent_").parse::<usize>() {
                            if let Some((name, _, _, _)) = state.phone_recents.get(idx) {
                                state.dial_number = name.clone();
                                state.phone_calling = true;
                                state.phone_call_duration = 0;
                                audio_engine.play_ringtone();
                            }
                        }
                    } else if id.starts_with("phone_call_contact_") {
                        if let Ok(idx) = id.trim_start_matches("phone_call_contact_").parse::<usize>() {
                            if let Some((name, num, _)) = state.phone_contacts.get(idx) {
                                state.dial_number = format!("{} ({})", name, num);
                                state.phone_calling = true;
                                state.phone_call_duration = 0;
                                audio_engine.play_ringtone();
                            }
                        }
                    } else if id.starts_with("phone_msg_contact_") {
                        if let Ok(idx) = id.trim_start_matches("phone_msg_contact_").parse::<usize>() {
                            if let Some((name, _, _)) = state.phone_contacts.get(idx) {
                                state.screen = Screen::AppMessages;
                                if let Some(pos) = state.sms_threads.iter().position(|t| t.name == *name) {
                                    state.sms_active_thread = Some(pos);
                                } else {
                                    state.sms_threads.insert(0, ChatThreadItem {
                                        name: name.clone(),
                                        avatar_char: name.chars().next().unwrap_or('👤').to_string(),
                                        messages: vec![
                                            ChatMessageItem {
                                                sender: name.clone(),
                                                text: "হ্যালো, কেমন আছেন?".into(),
                                                time: get_ist_time_str(),
                                                is_me: false,
                                            }
                                        ],
                                    });
                                    state.sms_active_thread = Some(0);
                                }
                            }
                        }
                    } else if id == "app_messages" {
                        state.sms_active_thread = None;
                        state.screen = Screen::AppMessages;
                    } else if id == "msg_thread_back" {
                        state.sms_active_thread = None;
                    } else if id == "msg_new" {
                        state.sms_threads.insert(0, ChatThreadItem {
                            name: "নতুন বার্তা".into(),
                            avatar_char: "ন".into(),
                            messages: vec![
                                ChatMessageItem {
                                    sender: "নতুন বার্তা".into(),
                                    text: "হ্যালো, কেমন আছেন?".into(),
                                    time: get_ist_time_str(),
                                    is_me: false,
                                }
                            ],
                        });
                        state.sms_active_thread = Some(0);
                    } else if id.starts_with("msg_thread_") {
                        if let Ok(idx) = id.trim_start_matches("msg_thread_").parse::<usize>() {
                            if idx < state.sms_threads.len() {
                                state.sms_active_thread = Some(idx);
                            }
                        }
                    } else if id == "msg_send" || id == "msg_quick_ok" || id == "msg_quick_coming" || id == "msg_quick_later" || id == "msg_quick_gotfile" {
                        let text = match id.as_str() {
                            "msg_quick_ok" => "ঠিক আছে 👍".to_string(),
                            "msg_quick_coming" => "আমি আসছি 🚗".to_string(),
                            "msg_quick_later" => "পরে বলি 📞".to_string(),
                            "msg_quick_gotfile" => "ফাইল পেয়েছি 📁".to_string(),
                            _ => {
                                if state.sms_reply_input.is_empty() {
                                    "ঠিক আছে, ধন্যবাদ!".to_string()
                                } else {
                                    std::mem::take(&mut state.sms_reply_input)
                                }
                            }
                        };
                        if let Some(t_idx) = state.sms_active_thread {
                            if let Some(th) = state.sms_threads.get_mut(t_idx) {
                                th.messages.push(ChatMessageItem {
                                    sender: "আমি".into(),
                                    text: text.clone(),
                                    time: get_ist_time_str(),
                                    is_me: true,
                                });
                                audio_engine.play_msg_sent_sound();
                                state.push_term_line(format!("[SMS] বার্তা প্রেরিত: {}", text), COLOR_GREEN);
                                state.trigger_sms_reply(t_idx, &text);
                            }
                        }
                    } else if id == "app_files" {
                        state.current_path = state.storage_root.join("home").join("joy").to_string_lossy().to_string();
                        state.screen = Screen::AppFiles;
                    } else if id == "bm_home" {
                        state.current_path = state.storage_root.join("home").join("joy").to_string_lossy().to_string();
                    } else if id == "bm_root" {
                        state.current_path = state.storage_root.to_string_lossy().to_string();
                    } else if id == "bm_docs" {
                        let docs = state.storage_root.join("home").join("joy").join("Documents");
                        let _ = std::fs::create_dir_all(&docs);
                        state.current_path = docs.to_string_lossy().to_string();
                    } else if id == "bm_up" {
                        let p = PathBuf::from(&state.current_path);
                        if p != state.storage_root && p.starts_with(&state.storage_root) {
                            if let Some(parent) = p.parent() {
                                if parent.starts_with(&state.storage_root) {
                                    state.current_path = parent.to_string_lossy().to_string();
                                }
                            }
                        }
                    } else if id == "file_new_folder" {
                        let new_dir = PathBuf::from(&state.current_path).join("নতুন_ফোল্ডার");
                        let _ = std::fs::create_dir_all(&new_dir);
                        state.push_term_line("[ফাইলস] নতুন ফোল্ডার সফলভাবে তৈরি হয়েছে।".into(), COLOR_GREEN);
                    } else if id == "file_new_file" {
                        let new_file = PathBuf::from(&state.current_path).join("নতুন_নথি.txt");
                        let _ = std::fs::write(&new_file, "# অনুরণ ওএস নতুন নথি\n\nএখানে আপনার বিষয়বস্তু লিখুন।\n");
                        state.push_term_line("[ফাইলস] নতুন নথি.txt সফলভাবে তৈরি হয়েছে।".into(), COLOR_GREEN);
                    } else if id.starts_with("file_open_") {
                        let name = id.trim_start_matches("file_open_");
                        let target = PathBuf::from(&state.current_path).join(name);
                        if target.is_dir() && target.starts_with(&state.storage_root) {
                            state.current_path = target.to_string_lossy().into();
                        }
                    } else if id.starts_with("file_act_") {
                        let name = id.trim_start_matches("file_act_");
                        let target = PathBuf::from(&state.current_path).join(name);
                        if name.ends_with(".mp4") || name.ends_with(".mkv") || name.ends_with(".avi") {
                            state.screen = Screen::AppMusic;
                        } else if name.ends_with(".mp3") || name.ends_with(".ogg") || name.ends_with(".flac") {
                            state.screen = Screen::AppMusic;
                        } else if name.ends_with(".txt") || name.ends_with(".md") || name.ends_with(".rs") || name.ends_with(".sh") || name.ends_with(".nil") {
                            if let Ok(content) = std::fs::read_to_string(&target) {
                                state.nano_lines = content.lines().map(String::from).collect();
                                if state.nano_lines.is_empty() { state.nano_lines.push(String::new()); }
                                state.nano_filename = name.into();
                                state.nano_row = 0;
                                state.nano_col = 0;
                                state.screen = Screen::NanoEditor;
                            }
                        }
                    } else if id == "app_settings" {
                        state.screen = Screen::AppSettings;
                    } else if id == "app_nilpkg" {
                        state.screen = Screen::AppNilPkg;
                    } else if id == "app_softbus" {
                        state.screen = Screen::AppSoftBus;
                    } else if id == "app_android" {
                        state.screen = Screen::AppAndroid;
                    } else if id == "app_terminal" {
                        state.screen = Screen::AppTerminal;
                    } else if id == "app_calc" || id == "app_calculator" {
                        state.screen = Screen::AppCalculator;
                    } else if id == "app_music" || id == "app_vlc" {
                        state.screen = Screen::AppMusic;
                    } else if id == "music_toggle_play" {
                        state.music_playing = !state.music_playing;
                        audio_engine.set_playing(state.music_playing);
                    } else if id == "music_next" {
                        state.music_track_idx = (state.music_track_idx + 1) % MUSIC_TRACKS.len();
                        state.music_progress = 0.0;
                        state.music_current_sec = 0;
                        state.music_playing = true;
                        audio_engine.set_track(state.music_track_idx);
                        audio_engine.set_playing(true);
                    } else if id == "music_prev" {
                        state.music_track_idx = if state.music_track_idx == 0 { MUSIC_TRACKS.len() - 1 } else { state.music_track_idx - 1 };
                        state.music_progress = 0.0;
                        state.music_current_sec = 0;
                        state.music_playing = true;
                        audio_engine.set_track(state.music_track_idx);
                        audio_engine.set_playing(true);
                    } else if id == "music_vol_down" {
                        state.volume_level = state.volume_level.saturating_sub(10);
                        audio_engine.set_volume(state.volume_level);
                    } else if id == "music_vol_up" {
                        state.volume_level = (state.volume_level + 10).min(100);
                        audio_engine.set_volume(state.volume_level);
                    } else if id == "music_scrub" {
                        let scrub_w = SCREEN_WIDTH as f32 - 32.0;
                        let ratio = ((px - 16) as f32 / scrub_w).clamp(0.0, 1.0);
                        state.music_progress = ratio;
                        let dur = MUSIC_TRACKS[state.music_track_idx % MUSIC_TRACKS.len()].duration_secs;
                        state.music_current_sec = (ratio * dur as f32) as u32;
                    } else if id.starts_with("music_select_") {
                        if let Ok(idx) = id.trim_start_matches("music_select_").parse::<usize>() {
                            state.music_track_idx = idx % MUSIC_TRACKS.len();
                            state.music_playing = true;
                            state.music_progress = 0.0;
                            state.music_current_sec = 0;
                            audio_engine.set_track(state.music_track_idx);
                            audio_engine.set_playing(true);
                        }
                    } else if id == "app_notes" {
                        state.notes_editing = false;
                        state.screen = Screen::AppNotes;
                    } else if id == "notes_new" {
                        state.notes_editing = true;
                        state.notes_edit_id = None;
                        state.notes_edit_title.clear();
                        state.notes_edit_content.clear();
                        state.notes_cursor_pos = 0;
                    } else if id.starts_with("note_open_") {
                        if let Ok(nid) = id.trim_start_matches("note_open_").parse::<usize>() {
                            if let Some(note) = state.notes.iter().find(|n| n.id == nid) {
                                state.notes_editing = true;
                                state.notes_edit_id = Some(note.id);
                                state.notes_edit_title = note.title.clone();
                                state.notes_edit_content = note.content.clone();
                                state.notes_cursor_pos = note.content.chars().count();
                            }
                        }
                    } else if id.starts_with("notes_cat_") {
                        let cat = id.trim_start_matches("notes_cat_");
                        state.notes_category = cat.to_string();
                    } else if id == "note_toggle_pin" {
                        if let Some(id_val) = state.notes_edit_id {
                            if let Some(n) = state.notes.iter_mut().find(|n| n.id == id_val) {
                                n.pinned = !n.pinned;
                            }
                        }
                    } else if id == "note_delete" {
                        if let Some(id_val) = state.notes_edit_id {
                            state.notes.retain(|n| n.id != id_val);
                            state.notes_editing = false;
                        }
                    } else if id == "notes_save" {
                        let title = if state.notes_edit_title.trim().is_empty() {
                            "নতুন নোট".to_string()
                        } else {
                            state.notes_edit_title.trim().to_string()
                        };
                        let time_str = get_ist_time_str();
                        if let Some(id_val) = state.notes_edit_id {
                            if let Some(n) = state.notes.iter_mut().find(|n| n.id == id_val) {
                                n.title = title.clone();
                                n.content = state.notes_edit_content.clone();
                                n.updated = time_str;
                            }
                        } else {
                            let next_id = state.notes.iter().map(|n| n.id).max().unwrap_or(0) + 1;
                            state.notes.insert(0, Note {
                                id: next_id,
                                title: title.clone(),
                                content: state.notes_edit_content.clone(),
                                category: state.notes_category.clone(),
                                updated: time_str,
                                pinned: false,
                                color: COLOR_GOLD,
                            });
                        }
                        // Write note to disk
                        let doc_dir = state.storage_root.join("home").join("joy").join("Documents");
                        let _ = std::fs::create_dir_all(&doc_dir);
                        let doc_file = doc_dir.join(format!("{}.txt", &title));
                        let _ = std::fs::write(&doc_file, &state.notes_edit_content);
                        state.notes_editing = false;
                    } else if id == "notes_cancel" {
                        state.notes_editing = false;
                    } else if id == "app_browser" {
                        state.screen = Screen::AppBrowser;
                    } else if id == "home_weather_card" {
                        state.screen = Screen::AppBrowser;
                        state.browser_url = "https://www.google.com/search?q=siliguri+weather".into();
                        state.browser_url_input = "https://www.google.com/search?q=siliguri+weather".into();
                        state.browser_is_editing_url = false;
                        #[cfg(target_os = "windows")]
                        if let Some(ref b) = embedded_browser {
                            let _ = nilbrowser::chromium::navigate_to(b, "https://www.google.com/search?q=siliguri+weather");
                        }
                    } else if id == "tab_add" {
                        state.new_browser_tab("https://www.google.com");
                        #[cfg(target_os = "windows")]
                        if let Some(ref b) = embedded_browser {
                            let _ = nilbrowser::chromium::navigate_to(b, "https://www.google.com");
                        }
                    } else if id.starts_with("tab_select_") {
                        if let Ok(idx) = id.trim_start_matches("tab_select_").parse::<usize>() {
                            state.switch_browser_tab(idx);
                            #[cfg(target_os = "windows")]
                            if let Some(ref b) = embedded_browser {
                                let _ = nilbrowser::chromium::navigate_to(b, &state.browser_url);
                            }
                        }
                    } else if id.starts_with("tab_close_") {
                        if let Ok(idx) = id.trim_start_matches("tab_close_").parse::<usize>() {
                            state.close_browser_tab(idx);
                            #[cfg(target_os = "windows")]
                            if let Some(ref b) = embedded_browser {
                                let _ = nilbrowser::chromium::navigate_to(b, &state.browser_url);
                            }
                        }
                    } else if id == "browser_reload" {
                        #[cfg(target_os = "windows")]
                        if let Some(ref b) = embedded_browser {
                            let _ = nilbrowser::chromium::reload(b);
                        }
                    } else if id == "browser_back" {
                        #[cfg(target_os = "windows")]
                        if let Some(ref b) = embedded_browser {
                            let _ = nilbrowser::chromium::go_back(b);
                        }
                    } else if id == "browser_go" {
                        state.browser_url = state.browser_url_input.clone();
                        state.browser_is_editing_url = false;
                        if state.browser_active_tab < state.browser_tabs.len() {
                            state.browser_tabs[state.browser_active_tab].url = state.browser_url.clone();
                        }
                        #[cfg(target_os = "windows")]
                        if let Some(ref b) = embedded_browser {
                            let _ = nilbrowser::chromium::navigate_to(b, &state.browser_url);
                        }
                    } else if id == "browser_url_bar" {
                        state.browser_is_editing_url = true;
                        state.browser_url_input = state.browser_url.clone();
                    } else if id.starts_with("calc_key_") {
                        let k = id.trim_start_matches("calc_key_");
                        state.exec_calc_press(k);
                    } else if id.starts_with("sb_drop_") {
                        let dev = id.trim_start_matches("sb_drop_");
                        state.push_term_line(format!("[সফটবাস] '{}' ডিভাইসে ফাইল সফলভাবে ড্রপ করা হয়েছে!", dev), COLOR_GREEN);
                    } else if id == "sb_scan" {
                        state.push_term_line("[সফটবাস] মেশ নেটওয়ার্কে ৩টি অ্যাক্টিভ ডিভাইস পাওয়া গেছে।".into(), COLOR_CYAN);
                    } else if id == "nano_act_save" {
                        state.nano_save();
                    } else if id == "nano_act_enter" {
                        state.nano_enter();
                    } else if id == "nano_act_exit" {
                        state.screen = Screen::AppTerminal;
                    } else if id.starts_with("lock_key_") {
                        let key = id.trim_start_matches("lock_key_");
                        if let Some(ch) = key.chars().next() {
                            if ch.is_ascii_digit() || (ch >= '০' && ch <= '৯') {
                                audio_engine.play_dtmf(ch);
                            }
                        }
                        if key == "<" {
                            state.pin_input.pop();
                        } else if key == "আনলক" || key == "UNLOCK" {
                            if state.pin_input == state.pin {
                                audio_engine.play_unlock_sound();
                                state.screen = Screen::Home;
                                state.pin_input.clear();
                                state.lock_error = false;
                            } else {
                                state.lock_error = true;
                            }
                        } else if state.pin_input.len() < 4 {
                            let digit_char = match key {
                                "০" => "0", "১" => "1", "২" => "2", "৩" => "3", "৪" => "4",
                                "৫" => "5", "৬" => "6", "৭" => "7", "৮" => "8", "৯" => "9",
                                other => other,
                            };
                            state.pin_input.push_str(digit_char);
                            if state.pin_input.len() == 4 && state.pin_input == state.pin {
                                audio_engine.play_unlock_sound();
                                state.screen = Screen::Home;
                                state.pin_input.clear();
                                state.lock_error = false;
                            }
                        }
                    } else if id.starts_with("dial_key_") {
                        let k = id.trim_start_matches("dial_key_");
                        if let Some(ch) = k.chars().next() {
                            audio_engine.play_dtmf(ch);
                        }
                        state.dial_number.push_str(k);
                    } else if id == "phone_del" {
                        state.dial_number.pop();
                    } else if id == "phone_call" {
                        state.phone_calling = true;
                        state.phone_call_duration = 0;
                        audio_engine.play_ringtone();
                    } else if id == "phone_end_call" {
                        state.phone_calling = false;
                    } else if id == "toggle_wifi" {
                        state.wifi_enabled = !state.wifi_enabled;
                    } else if id == "toggle_cellular" || id == "toggle_data" {
                        state.cellular_enabled = !state.cellular_enabled;
                        if state.cellular_enabled {
                            state.cellular_signal_level = 4;
                        } else {
                            state.cellular_signal_level = 0;
                        }
                    } else if id == "cycle_signal" {
                        if !state.cellular_enabled {
                            state.cellular_enabled = true;
                            state.cellular_signal_level = 4;
                        } else if state.cellular_signal_level > 1 {
                            state.cellular_signal_level -= 1;
                        } else if state.cellular_signal_level == 1 {
                            state.cellular_signal_level = 0;
                            state.cellular_enabled = false;
                        } else {
                            state.cellular_signal_level = 4;
                        }
                    } else if id == "toggle_battery_charge" {
                        state.battery_charging = !state.battery_charging;
                    } else if id == "toggle_bt" {
                        state.bt_enabled = !state.bt_enabled;
                    } else if id == "toggle_softbus" {
                        state.softbus_enabled = !state.softbus_enabled;
                    } else if id == "toggle_theme" {
                        state.dark_mode = !state.dark_mode;
                    } else if id == "toggle_torch" {
                        state.torch_enabled = !state.torch_enabled;
                    } else if id == "toggle_airplane" {
                        state.airplane_mode = !state.airplane_mode;
                        if state.airplane_mode {
                            state.cellular_enabled = false;
                            state.wifi_enabled = false;
                            state.cellular_signal_level = 0;
                        } else {
                            state.cellular_enabled = true;
                            state.wifi_enabled = true;
                            state.cellular_signal_level = 4;
                        }
                    } else if id == "toggle_hotspot" {
                        state.hotspot_enabled = !state.hotspot_enabled;
                    } else if id == "slider_brightness" {
                        let sw = SCREEN_WIDTH as f32 - 52.0;
                        let ratio = ((px - 26) as f32 / sw).clamp(0.1, 1.0);
                        state.brightness_level = (ratio * 100.0) as u8;
                    } else if id == "slider_volume" {
                        let sw = SCREEN_WIDTH as f32 - 52.0;
                        let ratio = ((px - 26) as f32 / sw).clamp(0.0, 1.0);
                        state.volume_level = (ratio * 100.0) as u8;
                        audio_engine.set_volume(state.volume_level);
                    } else if id.starts_with("term_chip_") {
                        let cmd = id.trim_start_matches("term_chip_").to_string();
                        if cmd == "nano" {
                            state.open_nano("my_note.txt");
                        } else if cmd == "vlc" {
                            state.screen = Screen::AppMusic;
                        } else if cmd == "nilpkg" {
                            state.exec_term_command("nilpkg list");
                        } else {
                            state.exec_term_command(&cmd);
                        }
                    } else if id.starts_with("pkg_act_") {
                        let pkg = id.trim_start_matches("pkg_act_");
                        if state.is_pkg_installed(pkg) {
                            if pkg == "org.videolan.vlc" {
                                state.screen = Screen::AppMusic;
                            } else if pkg == "org.mozilla.fenix" {
                                state.screen = Screen::AppBrowser;
                            } else {
                                state.push_term_line(format!("[+] অ্যাপ ওপেন হচ্ছে: {}", pkg), COLOR_GREEN);
                            }
                        } else {
                            state.install_pkg(pkg);
                        }
                    }
                }
            }
        }
        was_mouse_down = mouse_down;

        #[cfg(target_os = "windows")]
        if let Some(ref mut b) = embedded_browser {
            let should_be_visible = state.screen == Screen::AppBrowser;
            let _ = nilbrowser::chromium::set_visible(b, should_be_visible);
        }

        window.update_with_buffer(&buffer, SCREEN_WIDTH, SCREEN_HEIGHT)?;
    }

    Ok(())
}
