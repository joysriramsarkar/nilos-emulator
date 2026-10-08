// nilsim/src/audio.rs — Advanced Multi-Voice Polyphonic Synthesizer & Music Engine for Onuron OS
// Zero external dependencies; uses Win32 waveOut API on Windows with 44.1kHz 16-bit stereo PCM.

use std::sync::atomic::{AtomicBool, AtomicU8, AtomicUsize, Ordering};
use std::sync::Arc;
use std::thread;
use std::time::Duration;

#[repr(C)]
#[derive(Clone, Copy)]
struct WaveFormatEx {
    w_format_tag: u16,
    n_channels: u16,
    n_samples_per_sec: u32,
    n_avg_bytes_per_sec: u32,
    n_block_align: u16,
    w_bits_per_sample: u16,
    cb_size: u16,
}

#[repr(C)]
struct WaveHdr {
    lp_data: *mut u8,
    dw_buffer_length: u32,
    dw_bytes_recorded: u32,
    dw_user: usize,
    dw_flags: u32,
    dw_loops: u32,
    lp_next: *mut WaveHdr,
    reserved: usize,
}

#[cfg(target_os = "windows")]
#[link(name = "winmm")]
extern "system" {
    fn waveOutOpen(
        phwo: *mut isize,
        u_device_id: u32,
        pwfx: *const WaveFormatEx,
        dw_callback: usize,
        dw_instance: usize,
        fdw_open: u32,
    ) -> u32;
    fn waveOutPrepareHeader(hwo: isize, pwh: *mut WaveHdr, cbwh: u32) -> u32;
    fn waveOutWrite(hwo: isize, pwh: *mut WaveHdr, cbwh: u32) -> u32;
    fn waveOutUnprepareHeader(hwo: isize, pwh: *mut WaveHdr, cbwh: u32) -> u32;
    fn waveOutClose(hwo: isize) -> u32;
    fn waveOutReset(hwo: isize) -> u32;
}

// ─── Musical Melody Note Frequencies (Hz) ────────────────────────────────────
pub const NOTE_REST: f32 = 0.0;
pub const NOTE_C2: f32 = 65.41;
pub const NOTE_D2: f32 = 73.42;
pub const NOTE_E2: f32 = 82.41;
pub const NOTE_F2: f32 = 87.31;
pub const NOTE_G2: f32 = 98.00;
pub const NOTE_A2: f32 = 110.00;
pub const NOTE_B2: f32 = 123.47;
pub const NOTE_C3: f32 = 130.81;
pub const NOTE_CS3: f32 = 138.59;
pub const NOTE_D3: f32 = 146.83;
pub const NOTE_DS3: f32 = 155.56;
pub const NOTE_E3: f32 = 164.81;
pub const NOTE_F3: f32 = 174.61;
pub const NOTE_FS3: f32 = 185.00;
pub const NOTE_G3: f32 = 196.00;
pub const NOTE_GS3: f32 = 207.65;
pub const NOTE_A3: f32 = 220.00;
pub const NOTE_AS3: f32 = 233.08;
pub const NOTE_B3: f32 = 246.94;
pub const NOTE_C4: f32 = 261.63;
pub const NOTE_CS4: f32 = 277.18;
pub const NOTE_D4: f32 = 293.66;
pub const NOTE_DS4: f32 = 311.13;
pub const NOTE_E4: f32 = 329.63;
pub const NOTE_F4: f32 = 349.23;
pub const NOTE_FS4: f32 = 369.99;
pub const NOTE_G4: f32 = 392.00;
pub const NOTE_GS4: f32 = 415.30;
pub const NOTE_A4: f32 = 440.00;
pub const NOTE_AS4: f32 = 466.16;
pub const NOTE_B4: f32 = 493.88;
pub const NOTE_C5: f32 = 523.25;
pub const NOTE_CS5: f32 = 554.37;
pub const NOTE_D5: f32 = 587.33;
pub const NOTE_DS5: f32 = 622.25;
pub const NOTE_E5: f32 = 659.25;
pub const NOTE_F5: f32 = 698.46;
pub const NOTE_FS5: f32 = 739.99;
pub const NOTE_G5: f32 = 783.99;
pub const NOTE_GS5: f32 = 830.61;
pub const NOTE_A5: f32 = 880.00;
pub const NOTE_B5: f32 = 987.77;
pub const NOTE_C6: f32 = 1046.50;

// ─── Polyphonic Musical Tracks ───────────────────────────────────────────────

// Song 0: "আনন্দলোকে মঙ্গলালোকে" (Anandaloke Mongolaloke) — Rabindranath Tagore
const LEAD_ANANDALOKE: &[(f32, u32)] = &[
    (NOTE_G4, 400), (NOTE_G4, 200), (NOTE_A4, 400), (NOTE_G4, 400), (NOTE_E4, 600), (NOTE_REST, 100),
    (NOTE_G4, 400), (NOTE_G4, 200), (NOTE_A4, 400), (NOTE_G4, 400), (NOTE_E4, 600), (NOTE_REST, 100),
    (NOTE_G4, 300), (NOTE_A4, 300), (NOTE_C5, 500), (NOTE_B4, 300), (NOTE_A4, 400), (NOTE_G4, 600),
    (NOTE_F4, 300), (NOTE_E4, 300), (NOTE_D4, 500), (NOTE_E4, 300), (NOTE_C4, 800), (NOTE_REST, 300),
    (NOTE_E4, 400), (NOTE_G4, 400), (NOTE_C5, 600), (NOTE_D5, 300), (NOTE_C5, 500), (NOTE_B4, 400),
    (NOTE_A4, 400), (NOTE_G4, 400), (NOTE_E4, 600), (NOTE_D4, 400), (NOTE_C4, 800), (NOTE_REST, 400),
];
const CHORDS_ANANDALOKE: &[([f32; 3], u32)] = &[
    ([NOTE_C4, NOTE_E4, NOTE_G4], 2100),
    ([NOTE_C4, NOTE_E4, NOTE_G4], 2100),
    ([NOTE_F4, NOTE_A4, NOTE_C5], 2400),
    ([NOTE_G3, NOTE_B3, NOTE_D4], 2200),
    ([NOTE_C4, NOTE_E4, NOTE_G4], 2200),
    ([NOTE_F4, NOTE_A4, NOTE_C5], 2600),
];
const BASS_ANANDALOKE: &[(f32, u32)] = &[
    (NOTE_C3, 1050), (NOTE_G2, 1050),
    (NOTE_C3, 1050), (NOTE_G2, 1050),
    (NOTE_F2, 1200), (NOTE_C3, 1200),
    (NOTE_G2, 1100), (NOTE_C3, 1100),
    (NOTE_C3, 1100), (NOTE_E3, 1100),
    (NOTE_F2, 1300), (NOTE_C3, 1300),
];

// Song 1: "অনুরণ ওএস সিম্ফনি নং ১" (Onuron OS Symphony) — Synthwave & Electro
const LEAD_SYMPHONY: &[(f32, u32)] = &[
    (NOTE_E4, 250), (NOTE_GS4, 250), (NOTE_B4, 250), (NOTE_E5, 350), (NOTE_DS5, 250), (NOTE_B4, 250), (NOTE_GS4, 250),
    (NOTE_FS4, 250), (NOTE_A4, 250), (NOTE_CS5, 250), (NOTE_E5, 350), (NOTE_D5, 250), (NOTE_A4, 250), (NOTE_FS4, 250),
    (NOTE_G4, 250), (NOTE_B4, 250), (NOTE_D5, 250), (NOTE_G5, 400), (NOTE_FS5, 250), (NOTE_D5, 250), (NOTE_B4, 250),
    (NOTE_E5, 600), (NOTE_B4, 400), (NOTE_GS4, 400), (NOTE_E4, 800), (NOTE_REST, 300),
];
const CHORDS_SYMPHONY: &[([f32; 3], u32)] = &[
    ([NOTE_E4, NOTE_GS4, NOTE_B4], 1850),
    ([NOTE_D4, NOTE_FS4, NOTE_A4], 1850),
    ([NOTE_G4, NOTE_B4, NOTE_D5], 1900),
    ([NOTE_E4, NOTE_GS4, NOTE_B4], 2500),
];
const BASS_SYMPHONY: &[(f32, u32)] = &[
    (NOTE_E2, 462), (NOTE_E2, 462), (NOTE_E3, 462), (NOTE_E2, 462),
    (NOTE_D2, 462), (NOTE_D2, 462), (NOTE_D3, 462), (NOTE_D2, 462),
    (NOTE_G2, 475), (NOTE_G2, 475), (NOTE_G3, 475), (NOTE_G2, 475),
    (NOTE_E2, 625), (NOTE_B2, 625), (NOTE_E2, 625), (NOTE_E2, 625),
];

// Song 2: "লো-ফাই কোডিং বিটস" (Lo-Fi Coding Beats) — Rhodes 7th Chords
const LEAD_LOFI: &[(f32, u32)] = &[
    (NOTE_D4, 450), (NOTE_F4, 450), (NOTE_A4, 450), (NOTE_C5, 600),
    (NOTE_G3, 450), (NOTE_B3, 450), (NOTE_D4, 450), (NOTE_F4, 600),
    (NOTE_C4, 450), (NOTE_E4, 450), (NOTE_G4, 450), (NOTE_B4, 600),
    (NOTE_A3, 450), (NOTE_C4, 450), (NOTE_E4, 450), (NOTE_G4, 700),
    (NOTE_F4, 350), (NOTE_E4, 350), (NOTE_D4, 350), (NOTE_C4, 600), (NOTE_REST, 300),
];
const CHORDS_LOFI: &[([f32; 3], u32)] = &[
    ([NOTE_F4, NOTE_A4, NOTE_C5], 1950), // Dmin7 harmony
    ([NOTE_F4, NOTE_G4, NOTE_B4], 1950), // G7 harmony
    ([NOTE_E4, NOTE_G4, NOTE_B4], 1950), // Cmaj7 harmony
    ([NOTE_C4, NOTE_E4, NOTE_G4], 2050), // Amin7 harmony
    ([NOTE_D4, NOTE_F4, NOTE_A4], 1950),
];
const BASS_LOFI: &[(f32, u32)] = &[
    (NOTE_D2, 975), (NOTE_A2, 975),
    (NOTE_G2, 975), (NOTE_D3, 975),
    (NOTE_C2, 975), (NOTE_G2, 975),
    (NOTE_A2, 1025), (NOTE_E3, 1025),
    (NOTE_D2, 975), (NOTE_C2, 975),
];

// Song 3: "কারার ঐ লৌহকপাট" (Karar Oi Louhokopat) — Kazi Nazrul Islam March
const LEAD_NAZRUL: &[(f32, u32)] = &[
    (NOTE_E4, 350), (NOTE_E4, 350), (NOTE_G4, 400), (NOTE_A4, 500), (NOTE_B4, 600), (NOTE_REST, 100),
    (NOTE_B4, 350), (NOTE_C5, 350), (NOTE_B4, 350), (NOTE_A4, 400), (NOTE_G4, 600), (NOTE_REST, 100),
    (NOTE_E4, 350), (NOTE_G4, 350), (NOTE_A4, 500), (NOTE_B4, 500), (NOTE_C5, 700), (NOTE_REST, 100),
    (NOTE_D5, 400), (NOTE_C5, 300), (NOTE_B4, 400), (NOTE_A4, 400), (NOTE_G4, 500), (NOTE_E4, 800),
];
const CHORDS_NAZRUL: &[([f32; 3], u32)] = &[
    ([NOTE_E4, NOTE_G4, NOTE_B4], 2300),
    ([NOTE_A4, NOTE_C5, NOTE_E5], 2150),
    ([NOTE_E4, NOTE_G4, NOTE_B4], 2500),
    ([NOTE_D4, NOTE_FS4, NOTE_A4], 1500),
    ([NOTE_E4, NOTE_G4, NOTE_B4], 1300),
];
const BASS_NAZRUL: &[(f32, u32)] = &[
    (NOTE_E2, 575), (NOTE_E2, 575), (NOTE_B2, 575), (NOTE_E2, 575),
    (NOTE_A2, 537), (NOTE_A2, 537), (NOTE_E3, 537), (NOTE_A2, 537),
    (NOTE_E2, 625), (NOTE_E2, 625), (NOTE_B2, 625), (NOTE_E2, 625),
    (NOTE_D2, 750), (NOTE_B2, 750), (NOTE_E2, 1300),
];

// ─── Public Audio Engine Interface ───────────────────────────────────────────

pub struct AudioEngine {
    is_playing: Arc<AtomicBool>,
    track_idx: Arc<AtomicUsize>,
    volume: Arc<AtomicU8>,
    sfx_trigger: Arc<AtomicU8>, // 1 = click, 2..13 = DTMF 0-9,*,#, 14 = ringtone, 15 = lock, 16 = unlock, 17 = msg_sent, 18 = notification
    spectrum: Arc<[AtomicU8; 18]>,
}

impl AudioEngine {
    pub fn new() -> Self {
        let is_playing = Arc::new(AtomicBool::new(true));
        let track_idx = Arc::new(AtomicUsize::new(0));
        let volume = Arc::new(AtomicU8::new(80));
        let sfx_trigger = Arc::new(AtomicU8::new(0));

        let mut spec_arr: [AtomicU8; 18] = Default::default();
        for item in spec_arr.iter_mut() {
            *item = AtomicU8::new(0);
        }
        let spectrum = Arc::new(spec_arr);

        let engine = AudioEngine {
            is_playing: Arc::clone(&is_playing),
            track_idx: Arc::clone(&track_idx),
            volume: Arc::clone(&volume),
            sfx_trigger: Arc::clone(&sfx_trigger),
            spectrum: Arc::clone(&spectrum),
        };

        #[cfg(target_os = "windows")]
        {
            let is_p = Arc::clone(&is_playing);
            let trk = Arc::clone(&track_idx);
            let vol = Arc::clone(&volume);
            let sfx = Arc::clone(&sfx_trigger);
            let spec = Arc::clone(&spectrum);

            thread::Builder::new()
                .name("onuron_audio_dsp".to_string())
                .spawn(move || {
                    run_win32_dsp_loop(is_p, trk, vol, sfx, spec);
                })
                .expect("Failed to spawn audio thread");
        }

        engine
    }

    pub fn set_playing(&self, playing: bool) {
        self.is_playing.store(playing, Ordering::Relaxed);
    }

    pub fn set_track(&self, idx: usize) {
        self.track_idx.store(idx, Ordering::Relaxed);
    }

    pub fn set_volume(&self, vol: u8) {
        self.volume.store(vol.min(100), Ordering::Relaxed);
    }

    pub fn play_click(&self) {
        self.sfx_trigger.store(1, Ordering::Relaxed);
    }

    pub fn play_dtmf(&self, digit: char) {
        let code = match digit {
            '0' | '০' => 2,
            '1' | '১' => 3,
            '2' | '২' => 4,
            '3' | '৩' => 5,
            '4' | '৪' => 6,
            '5' | '৫' => 7,
            '6' | '৬' => 8,
            '7' | '৭' => 9,
            '8' | '৮' => 10,
            '9' | '৯' => 11,
            '*' => 12,
            '#' => 13,
            _ => 1,
        };
        self.sfx_trigger.store(code, Ordering::Relaxed);
    }

    pub fn play_ringtone(&self) {
        self.sfx_trigger.store(14, Ordering::Relaxed);
    }

    pub fn play_lock_sound(&self) {
        self.sfx_trigger.store(15, Ordering::Relaxed);
    }

    pub fn play_unlock_sound(&self) {
        self.sfx_trigger.store(16, Ordering::Relaxed);
    }

    pub fn play_msg_sent_sound(&self) {
        self.sfx_trigger.store(17, Ordering::Relaxed);
    }

    pub fn get_spectrum(&self) -> [u8; 18] {
        let mut out = [0u8; 18];
        for i in 0..18 {
            out[i] = self.spectrum[i].load(Ordering::Relaxed);
        }
        out
    }
}

// ─── Real-Time Polyphonic DSP Engine (Win32 waveOut) ──────────────────────────

#[cfg(target_os = "windows")]
fn run_win32_dsp_loop(
    is_playing: Arc<AtomicBool>,
    track_idx: Arc<AtomicUsize>,
    volume: Arc<AtomicU8>,
    sfx_trigger: Arc<AtomicU8>,
    spectrum: Arc<[AtomicU8; 18]>,
) {
    const SAMPLE_RATE: u32 = 44100;
    const CHANNELS: u16 = 2;
    const BUFFER_SAMPLES: usize = 2048;
    const DELAY_SAMPLES: usize = 6000; // ~136ms stereo echo delay buffer

    let wfx = WaveFormatEx {
        w_format_tag: 1, // WAVE_FORMAT_PCM
        n_channels: CHANNELS,
        n_samples_per_sec: SAMPLE_RATE,
        n_avg_bytes_per_sec: SAMPLE_RATE * (CHANNELS as u32) * 2,
        n_block_align: CHANNELS * 2,
        w_bits_per_sample: 16,
        cb_size: 0,
    };

    let mut h_wave_out: isize = 0;
    let res = unsafe {
        waveOutOpen(
            &mut h_wave_out,
            0xFFFFFFFF, // WAVE_MAPPER
            &wfx,
            0,
            0,
            0,
        )
    };

    if res != 0 {
        eprintln!("[Audio] Warning: waveOutOpen failed with error code: {}", res);
        return;
    } else {
        println!("[Audio] Win32 waveOut device initialized (44.1kHz Stereo PCM, 4-buffer ring)");
    }

    // 4-buffer ring queue for continuous, glitch-free playback
    const NUM_BUFFERS: usize = 4;
    let mut raw_bufs: Vec<Vec<i16>> = (0..NUM_BUFFERS)
        .map(|_| vec![0i16; BUFFER_SAMPLES * CHANNELS as usize])
        .collect();

    let mut hdrs: Vec<WaveHdr> = raw_bufs
        .iter_mut()
        .map(|buf| WaveHdr {
            lp_data: buf.as_mut_ptr() as *mut u8,
            dw_buffer_length: (buf.len() * 2) as u32,
            dw_bytes_recorded: 0,
            dw_user: 0,
            dw_flags: 0,
            dw_loops: 0,
            lp_next: std::ptr::null_mut(),
            reserved: 0,
        })
        .collect();

    for hdr in hdrs.iter_mut() {
        unsafe {
            waveOutPrepareHeader(h_wave_out, hdr, std::mem::size_of::<WaveHdr>() as u32);
        }
    }

    let mut was_submitted = [false; NUM_BUFFERS];
    let mut active_buffer = 0usize;
    let mut last_track = usize::MAX;

    // Voice 1: Lead Melody
    let mut lead_idx = 0usize;
    let mut lead_samples_left = 0usize;
    let mut lead_phase = 0.0f32;
    let mut lead_freq = 0.0f32;
    let mut lead_target_freq = 0.0f32;

    // Voice 2: Chords Harmony Triad
    let mut chord_idx = 0usize;
    let mut chord_samples_left = 0usize;
    let mut chord_phases = [0.0f32; 3];
    let mut chord_freqs = [0.0f32; 3];

    // Voice 3: Bassline
    let mut bass_idx = 0usize;
    let mut bass_samples_left = 0usize;
    let mut bass_phase = 0.0f32;
    let mut bass_freq = 0.0f32;

    // Voice 4: Percussion / Beat / Drone
    let mut drum_samples_left = 0usize;
    let mut drum_type = 0u8; // 1 = kick, 2 = snare, 3 = hihat
    let mut drum_phase = 0.0f32;
    let mut beat_tick = 0usize;

    // Reverb / Stereo Comb Delay Buffer
    let mut delay_buf_l = vec![0.0f32; DELAY_SAMPLES];
    let mut delay_buf_r = vec![0.0f32; DELAY_SAMPLES];
    let mut delay_pos = 0usize;

    // SFX synth state
    let mut sfx_samples_left = 0usize;
    let mut sfx_freq1 = 0.0f32;
    let mut sfx_freq2 = 0.0f32;
    let mut sfx_phase1 = 0.0f32;
    let mut sfx_phase2 = 0.0f32;
    let mut sfx_type = 0u8; // 0 = generic sine, 1 = chime, 2 = noise/whoosh

    // Noise PRNG
    let mut noise_seed: u32 = 0x12345678;

    while h_wave_out != 0 {
        // Check for SFX trigger
        let trigger = sfx_trigger.swap(0, Ordering::Relaxed);
        if trigger > 0 {
            sfx_type = 0;
            match trigger {
                1 => {
                    // UI Click (Snappy wooden tactile click)
                    sfx_freq1 = 1200.0;
                    sfx_freq2 = 2400.0;
                    sfx_samples_left = (SAMPLE_RATE as f32 * 0.015) as usize; // 15ms
                }
                2..=13 => {
                    // DTMF standard tones
                    let (f1, f2) = match trigger {
                        3 => (697.0, 1209.0), // 1
                        4 => (697.0, 1336.0), // 2
                        5 => (697.0, 1477.0), // 3
                        6 => (770.0, 1209.0), // 4
                        7 => (770.0, 1336.0), // 5
                        8 => (770.0, 1477.0), // 6
                        9 => (852.0, 1209.0), // 7
                        10 => (852.0, 1336.0), // 8
                        11 => (852.0, 1477.0), // 9
                        12 => (941.0, 1209.0), // *
                        2 => (941.0, 1336.0),  // 0
                        13 => (941.0, 1477.0), // #
                        _ => (697.0, 1209.0),
                    };
                    sfx_freq1 = f1;
                    sfx_freq2 = f2;
                    sfx_samples_left = (SAMPLE_RATE as f32 * 0.10) as usize; // 100ms
                }
                14 => {
                    // Phone Ringtone (440Hz + 480Hz dial-in chord)
                    sfx_freq1 = 440.0;
                    sfx_freq2 = 480.0;
                    sfx_samples_left = (SAMPLE_RATE as f32 * 0.45) as usize; // 450ms
                }
                15 => {
                    // Device Lock (Mechanical click latch)
                    sfx_freq1 = 900.0;
                    sfx_freq2 = 450.0;
                    sfx_samples_left = (SAMPLE_RATE as f32 * 0.04) as usize; // 40ms
                }
                16 => {
                    // Device Unlock (Ascending Material You chime)
                    sfx_freq1 = 523.25; // C5
                    sfx_freq2 = 659.25; // E5
                    sfx_samples_left = (SAMPLE_RATE as f32 * 0.18) as usize;
                    sfx_type = 1;
                }
                17 => {
                    // SMS Sent / Whoosh Pop
                    sfx_freq1 = 880.0;
                    sfx_freq2 = 1760.0;
                    sfx_samples_left = (SAMPLE_RATE as f32 * 0.08) as usize;
                    sfx_type = 2;
                }
                _ => {}
            }
        }

        let playing = is_playing.load(Ordering::Relaxed);
        let cur_track = track_idx.load(Ordering::Relaxed) % 4;

        if cur_track != last_track {
            last_track = cur_track;
            lead_idx = 0;
            lead_samples_left = 0;
            lead_phase = 0.0;
            lead_freq = 0.0;
            chord_idx = 0;
            chord_samples_left = 0;
            chord_phases = [0.0; 3];
            bass_idx = 0;
            bass_samples_left = 0;
            bass_phase = 0.0;
            beat_tick = 0;
            drum_samples_left = 0;
        }

        let master_vol = (volume.load(Ordering::Relaxed) as f32 / 100.0).clamp(0.0, 1.0);

        let (lead_arr, chords_arr, bass_arr) = match cur_track {
            0 => (LEAD_ANANDALOKE, CHORDS_ANANDALOKE, BASS_ANANDALOKE),
            1 => (LEAD_SYMPHONY, CHORDS_SYMPHONY, BASS_SYMPHONY),
            2 => (LEAD_LOFI, CHORDS_LOFI, BASS_LOFI),
            _ => (LEAD_NAZRUL, CHORDS_NAZRUL, BASS_NAZRUL),
        };

        if was_submitted[active_buffer] {
            while (hdrs[active_buffer].dw_flags & 1) == 0 {
                thread::sleep(Duration::from_millis(1));
            }
        }

        let active_slice = &mut raw_bufs[active_buffer];

        // Track live spectral energy bins for UI visualizer
        let mut band_energy = [0.0f32; 18];

        for frame_idx in 0..BUFFER_SAMPLES {
            // ── 1. Update Lead Melody ──────────────────────────────────────────
            if lead_samples_left == 0 {
                if playing {
                    let (freq, ms) = lead_arr[lead_idx % lead_arr.len()];
                    lead_target_freq = freq;
                    lead_samples_left = ((SAMPLE_RATE as f32) * (ms as f32 / 1000.0)) as usize;
                    lead_idx += 1;
                } else {
                    lead_target_freq = 0.0;
                    lead_samples_left = 1024;
                }
            }
            lead_samples_left = lead_samples_left.saturating_sub(1);
            // Smooth glide (Portamento)
            lead_freq += (lead_target_freq - lead_freq) * 0.02;

            // ── 2. Update Chords Harmony ───────────────────────────────────────
            if chord_samples_left == 0 {
                if playing {
                    let (triad, ms) = chords_arr[chord_idx % chords_arr.len()];
                    chord_freqs = triad;
                    chord_samples_left = ((SAMPLE_RATE as f32) * (ms as f32 / 1000.0)) as usize;
                    chord_idx += 1;
                } else {
                    chord_freqs = [0.0; 3];
                    chord_samples_left = 1024;
                }
            }
            chord_samples_left = chord_samples_left.saturating_sub(1);

            // ── 3. Update Bassline ─────────────────────────────────────────────
            if bass_samples_left == 0 {
                if playing {
                    let (freq, ms) = bass_arr[bass_idx % bass_arr.len()];
                    bass_freq = freq;
                    bass_samples_left = ((SAMPLE_RATE as f32) * (ms as f32 / 1000.0)) as usize;
                    bass_idx += 1;
                } else {
                    bass_freq = 0.0;
                    bass_samples_left = 1024;
                }
            }
            bass_samples_left = bass_samples_left.saturating_sub(1);

            // ── 4. Synthesize Lead Voice (With Vibrato & Tube Harmonics) ────────
            let mut lead_sample = 0.0f32;
            if playing && lead_freq > 15.0 {
                // 5.5 Hz subtle vibrato
                let vib = ((beat_tick as f32 * 5.5 * 2.0 * std::f32::consts::PI) / (SAMPLE_RATE as f32)).sin() * 2.5;
                let actual_lead_f = (lead_freq + vib).max(10.0);

                let step = (actual_lead_f * 2.0 * std::f32::consts::PI) / (SAMPLE_RATE as f32);
                lead_phase = (lead_phase + step) % (2.0 * std::f32::consts::PI);

                let h1 = lead_phase.sin();
                let h2 = (lead_phase * 2.0).sin() * 0.4;
                let h3 = (lead_phase * 3.0).sin() * 0.15;
                let h4 = (lead_phase * 4.0).sin() * 0.05;

                // Smooth ADSR envelope
                let env = if lead_samples_left < 300 {
                    (lead_samples_left as f32) / 300.0
                } else {
                    1.0f32
                };
                lead_sample = (h1 + h2 + h3 + h4) * 0.22 * env;
            }

            // ── 5. Synthesize Chords Voice (Warm Lush Electric Piano Pad) ──────
            let mut chord_sample_l = 0.0f32;
            let mut chord_sample_r = 0.0f32;
            if playing {
                let chord_env = if chord_samples_left < 400 {
                    (chord_samples_left as f32) / 400.0
                } else {
                    1.0f32
                };
                for (i, f) in chord_freqs.iter().enumerate() {
                    if *f > 20.0 {
                        let step = (*f * 2.0 * std::f32::consts::PI) / (SAMPLE_RATE as f32);
                        chord_phases[i] = (chord_phases[i] + step) % (2.0 * std::f32::consts::PI);
                        let s = chord_phases[i].sin() * 0.08 * chord_env;
                        if i == 0 {
                            chord_sample_l += s * 1.1;
                            chord_sample_r += s * 0.6;
                        } else if i == 1 {
                            chord_sample_l += s * 0.8;
                            chord_sample_r += s * 0.8;
                        } else {
                            chord_sample_l += s * 0.6;
                            chord_sample_r += s * 1.1;
                        }
                    }
                }
            }

            // ── 6. Synthesize Bass Voice (Deep Warm Sub-bass) ───────────────────
            let mut bass_sample = 0.0f32;
            if playing && bass_freq > 20.0 {
                let step = (bass_freq * 2.0 * std::f32::consts::PI) / (SAMPLE_RATE as f32);
                bass_phase = (bass_phase + step) % (2.0 * std::f32::consts::PI);
                let b1 = bass_phase.sin();
                let b2 = (bass_phase * 2.0).sin() * 0.25;
                let bass_env = if bass_samples_left < 200 {
                    (bass_samples_left as f32) / 200.0
                } else {
                    1.0f32
                };
                bass_sample = (b1 + b2) * 0.18 * bass_env;
            }

            // ── 7. Synthesize Rhythm / Drum Percussion ───────────────────────────
            beat_tick += 1;
            let tempo_samples = (SAMPLE_RATE as f32 * 0.25) as usize; // 16th beat pulse
            if beat_tick % tempo_samples == 0 && playing {
                let step_in_bar = (beat_tick / tempo_samples) % 16;
                if cur_track == 1 || cur_track == 3 {
                    // Synthwave / March beat: Kick on 0, 4, 8, 12, Snare on 4, 12
                    if step_in_bar == 0 || step_in_bar == 8 {
                        drum_type = 1; // Kick
                        drum_samples_left = 2200;
                        drum_phase = 0.0;
                    } else if step_in_bar == 4 || step_in_bar == 12 {
                        drum_type = 2; // Snare
                        drum_samples_left = 3000;
                    } else if step_in_bar % 2 == 0 {
                        drum_type = 3; // Hi-hat
                        drum_samples_left = 800;
                    }
                } else if cur_track == 2 {
                    // Lo-fi Boom-bap chill beat
                    if step_in_bar == 0 || step_in_bar == 6 || step_in_bar == 10 {
                        drum_type = 1; // Soft Kick
                        drum_samples_left = 1800;
                        drum_phase = 0.0;
                    } else if step_in_bar == 4 || step_in_bar == 12 {
                        drum_type = 2; // Rimshot
                        drum_samples_left = 2200;
                    } else if step_in_bar % 2 == 0 {
                        drum_type = 3; // Soft Hat
                        drum_samples_left = 600;
                    }
                } else {
                    // Tanpura Drone / Soft Tabla Tap for Rabindrasangeet
                    if step_in_bar == 0 || step_in_bar == 8 {
                        drum_type = 1; // Bayan bass tap
                        drum_samples_left = 1500;
                        drum_phase = 0.0;
                    }
                }
            }

            let mut drum_sample = 0.0f32;
            if drum_samples_left > 0 {
                drum_samples_left -= 1;
                // PRNG Noise
                noise_seed = noise_seed.wrapping_mul(1664525).wrapping_add(1013904223);
                let white_noise = ((noise_seed >> 16) as f32 / 32768.0) - 1.0;

                if drum_type == 1 {
                    // Pitch drop Kick 80Hz -> 35Hz
                    let pitch = 35.0 + (drum_samples_left as f32 / 2200.0) * 65.0;
                    let step = (pitch * 2.0 * std::f32::consts::PI) / (SAMPLE_RATE as f32);
                    drum_phase += step;
                    let env = (drum_samples_left as f32 / 2200.0).powf(1.8);
                    drum_sample = drum_phase.sin() * 0.28 * env;
                } else if drum_type == 2 {
                    // Snare: Noise + 180Hz body
                    let step = (180.0 * 2.0 * std::f32::consts::PI) / (SAMPLE_RATE as f32);
                    drum_phase += step;
                    let env = (drum_samples_left as f32 / 3000.0).powf(1.5);
                    drum_sample = (white_noise * 0.15 + drum_phase.sin() * 0.12) * env;
                } else if drum_type == 3 {
                    // Hi-hat: filtered noise
                    let env = (drum_samples_left as f32 / 800.0).powf(2.0);
                    drum_sample = white_noise * 0.06 * env;
                }
            }

            // ── 8. Synthesize SFX Sample ─────────────────────────────────────────
            let mut sfx_sample = 0.0f32;
            if sfx_samples_left > 0 {
                sfx_samples_left -= 1;
                let step1 = (sfx_freq1 * 2.0 * std::f32::consts::PI) / (SAMPLE_RATE as f32);
                let step2 = (sfx_freq2 * 2.0 * std::f32::consts::PI) / (SAMPLE_RATE as f32);
                sfx_phase1 = (sfx_phase1 + step1) % (2.0 * std::f32::consts::PI);
                sfx_phase2 = (sfx_phase2 + step2) % (2.0 * std::f32::consts::PI);

                if sfx_type == 1 {
                    // Chime
                    sfx_sample = (sfx_phase1.sin() * 0.35 + sfx_phase2.sin() * 0.35) * master_vol.max(0.5);
                } else if sfx_type == 2 {
                    // Pop / Whoosh
                    noise_seed = noise_seed.wrapping_mul(1664525).wrapping_add(1013904223);
                    let white_noise = ((noise_seed >> 16) as f32 / 32768.0) - 1.0;
                    sfx_sample = (sfx_phase1.sin() * 0.25 + white_noise * 0.1) * master_vol.max(0.5);
                } else {
                    sfx_sample = (sfx_phase1.sin() * 0.35 + sfx_phase2.sin() * 0.35) * master_vol.max(0.4);
                }
            }

            // ── 9. Spatial Stereo Delay & Reverb Mixing ─────────────────────────
            let dry_left = (lead_sample * 0.9 + chord_sample_l + bass_sample + drum_sample) * master_vol;
            let dry_right = (lead_sample * 0.9 + chord_sample_r + bass_sample + drum_sample) * master_vol;

            // Read from circular delay buffer
            let delayed_l = delay_buf_l[delay_pos];
            let delayed_r = delay_buf_r[delay_pos];

            // Feedback and write
            delay_buf_l[delay_pos] = dry_left * 0.22 + delayed_r * 0.18;
            delay_buf_r[delay_pos] = dry_right * 0.22 + delayed_l * 0.18;
            delay_pos = (delay_pos + 1) % DELAY_SAMPLES;

            let wet_left = (dry_left + delayed_l * 0.3 + sfx_sample).clamp(-0.95, 0.95);
            let wet_right = (dry_right + delayed_r * 0.3 + sfx_sample).clamp(-0.95, 0.95);

            let sample_l_i16 = (wet_left * 32767.0) as i16;
            let sample_r_i16 = (wet_right * 32767.0) as i16;

            active_slice[frame_idx * 2] = sample_l_i16;     // Left channel
            active_slice[frame_idx * 2 + 1] = sample_r_i16; // Right channel

            // Accumulate frequency energy into 18 bands for the visualizer
            let abs_mag = (wet_left.abs() + wet_right.abs()) * 0.5;
            let target_band = if lead_freq > 20.0 {
                let log_f = lead_freq.log2();
                let b = ((log_f - 6.0) * 3.2).clamp(0.0, 17.0) as usize;
                b
            } else {
                frame_idx % 18
            };
            band_energy[target_band] += abs_mag;
            band_energy[0] += bass_sample.abs() * 1.5;
            band_energy[1] += drum_sample.abs() * 1.2;
        }

        // Send to Win32 waveOut
        unsafe {
            waveOutWrite(h_wave_out, &mut hdrs[active_buffer], std::mem::size_of::<WaveHdr>() as u32);
        }
        was_submitted[active_buffer] = true;

        // Export smooth live spectrum levels (0..26 height) to atomics for UI
        for (i, bin) in band_energy.iter().enumerate() {
            let val = if playing {
                ((bin * 32.0 / (BUFFER_SAMPLES as f32) * 25.0) as u8).min(26).max(3)
            } else {
                2u8
            };
            spectrum[i].store(val, Ordering::Relaxed);
        }

        active_buffer = (active_buffer + 1) % NUM_BUFFERS;
    }

    unsafe {
        waveOutReset(h_wave_out);
        waveOutClose(h_wave_out);
    }
}
