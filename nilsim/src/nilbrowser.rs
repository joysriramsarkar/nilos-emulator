// NilOS WebView2 (Evergreen Chromium) Embedded Browser
// Uses Microsoft WebView2 COM API to embed the real Chromium/V8 engine inside the NilOS phone frame
#![allow(dead_code, unused_variables, unused_imports)]

#[cfg(target_os = "windows")]
pub mod chromium {
    use std::sync::mpsc;

    use webview2_com::{
        CoreWebView2EnvironmentOptions,
        CreateCoreWebView2ControllerCompletedHandler,
        CreateCoreWebView2EnvironmentCompletedHandler,
        Microsoft::Web::WebView2::Win32::{
            CreateCoreWebView2EnvironmentWithOptions, ICoreWebView2, ICoreWebView2Controller,
            ICoreWebView2Environment, ICoreWebView2EnvironmentOptions,
        },
        wait_with_pump,
    };
    use windows::{
        core::PWSTR,
        Win32::{
            Foundation::{HWND, RECT, E_POINTER},
            System::Com::{CoInitializeEx, COINIT_APARTMENTTHREADED},
        },
    };

    pub struct NilBrowser {
        pub controller: ICoreWebView2Controller,
        pub webview: ICoreWebView2,
        pub is_visible: bool,
    }

    unsafe impl Send for NilBrowser {}
    unsafe impl Sync for NilBrowser {}

    pub fn create_embedded_browser(
        parent_hwnd: isize,
        x: i32,
        y: i32,
        w: i32,
        h: i32,
        url: &str,
    ) -> windows::core::Result<NilBrowser> {
        unsafe {
            let _ = CoInitializeEx(None, COINIT_APARTMENTTHREADED);
        }

        let parent = HWND(parent_hwnd as *mut _);
        let url_str = url.to_string();

        // Create WebView2 Environment with modern overlay scrollbars
        let env = create_environment()?;

        // Create Controller embedded into parent HWND
        let controller = create_controller(parent, &env)?;

        // Set bounds: position at (x, y, w, h) inside parent window
        let bounds = RECT {
            left: x,
            top: y,
            right: x + w,
            bottom: y + h,
        };
        unsafe {
            controller.SetBounds(bounds)?;
            controller.SetIsVisible(true)?;
        }

        // Get ICoreWebView2 and navigate to URL
        let webview = unsafe { controller.CoreWebView2()? };
        let mut url_wide: Vec<u16> = url_str.encode_utf16().chain(std::iter::once(0)).collect();
        unsafe {
            webview.Navigate(PWSTR(url_wide.as_mut_ptr()))?;
        }

        Ok(NilBrowser {
            controller,
            webview,
            is_visible: true,
        })
    }

    pub fn set_bounds(
        browser: &NilBrowser,
        x: i32,
        y: i32,
        w: i32,
        h: i32,
    ) -> windows::core::Result<()> {
        let bounds = RECT {
            left: x,
            top: y,
            right: x + w,
            bottom: y + h,
        };
        unsafe {
            browser.controller.SetBounds(bounds)?;
        }
        Ok(())
    }

    pub fn set_visible(browser: &mut NilBrowser, visible: bool) -> windows::core::Result<()> {
        if browser.is_visible != visible {
            browser.is_visible = visible;
            unsafe {
                browser.controller.SetIsVisible(visible)?;
            }
        }
        Ok(())
    }

    pub fn navigate_to(browser: &NilBrowser, url: &str) -> windows::core::Result<()> {
        let mut url_wide: Vec<u16> = url.encode_utf16().chain(std::iter::once(0)).collect();
        unsafe {
            browser.webview.Navigate(PWSTR(url_wide.as_mut_ptr()))?;
        }
        Ok(())
    }

    pub fn reload(browser: &NilBrowser) -> windows::core::Result<()> {
        unsafe {
            browser.webview.Reload()?;
        }
        Ok(())
    }

    pub fn go_back(browser: &NilBrowser) -> windows::core::Result<()> {
        unsafe {
            browser.webview.GoBack()?;
        }
        Ok(())
    }

    pub fn go_forward(browser: &NilBrowser) -> windows::core::Result<()> {
        unsafe {
            browser.webview.GoForward()?;
        }
        Ok(())
    }

    fn create_environment() -> windows::core::Result<ICoreWebView2Environment> {
        let (tx, rx) = mpsc::channel();
        let handler = CreateCoreWebView2EnvironmentCompletedHandler::create(Box::new(
            move |_err, env: Option<ICoreWebView2Environment>| {
                let result = env.ok_or_else(|| windows::core::Error::from(E_POINTER));
                let _ = tx.send(result);
                Ok(())
            },
        ));

        let options = CoreWebView2EnvironmentOptions::default();
        unsafe {
            // Enable Overlay / Mobile slim scrollbars & touch emulation
            options.set_additional_browser_arguments(
                "--enable-features=OverlayScrollbar,FluentOverlayScrollbars --enable-blink-features=ScrollbarColor".to_string(),
            );
        }
        let env_options: ICoreWebView2EnvironmentOptions = options.into();

        unsafe {
            CreateCoreWebView2EnvironmentWithOptions(None, None, &env_options, &handler)?;
        }
        wait_with_pump(rx).map_err(|_| windows::core::Error::from(E_POINTER))?
    }

    fn create_controller(
        hwnd: HWND,
        env: &ICoreWebView2Environment,
    ) -> windows::core::Result<ICoreWebView2Controller> {
        let (tx, rx) = mpsc::channel();
        let handler = CreateCoreWebView2ControllerCompletedHandler::create(Box::new(
            move |_err, ctl: Option<ICoreWebView2Controller>| {
                let result = ctl.ok_or_else(|| windows::core::Error::from(E_POINTER));
                let _ = tx.send(result);
                Ok(())
            },
        ));
        unsafe {
            env.CreateCoreWebView2Controller(hwnd, &handler)?;
        }
        wait_with_pump(rx).map_err(|_| windows::core::Error::from(E_POINTER))?
    }
}
