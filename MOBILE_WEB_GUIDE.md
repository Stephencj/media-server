# Mobile Web Redesign - Comprehensive Testing and Deployment Guide

## 1. Overview

### What Was Implemented Across 5 Phases

This guide documents a complete mobile-first redesign of the Media Server web interface, rolled out in five coordinated phases. The implementation provides a modern, touch-friendly experience for mobile browsers while maintaining full desktop compatibility.

**Phase 1: Core Responsiveness**
- Viewport configuration and responsive typography
- CSS custom properties for adaptive spacing (44px touch targets minimum)
- Flexible grid layouts that adapt from 320px (iPhone SE) to 1400px+ (desktop)
- Safe area inset support for notched devices
- Touch-friendly form inputs (16px font size to prevent iOS zoom)
- Modal and dialog layouts that stack properly on mobile

**Phase 2: Navigation Structure**
- Bottom navigation bar (visible on mobile <768px)
- Navigation drawers for Browse (Movies/Shows/Extras) and More (Sections/Admin/Logout)
- Drawer slide-in animations with backdrop tap-to-close
- Swipe-down gesture support to close drawers
- Active state indicators and visual feedback
- Desktop navigation bar (visible on ≥768px)

**Phase 3: Touch Interactions**
- Always-visible play buttons on touch devices (no hover-dependent interaction)
- Long-press activation (500ms) for playlist drag functionality
- Smooth playlist item reordering with visual feedback
- Haptic feedback integration (iOS devices)
- Swipe-down modal dismiss gesture
- Touch feedback on all interactive elements

**Phase 4: Enhanced Features**
- Pull-to-refresh on Home, Movies, Shows, and Extras views
- Double-tap video player controls (left 10s rewind, right 10s forward)
- Single-tap video control toggle (show/hide controls)
- Volume control swipe gestures
- Fullscreen button functionality
- Picture-in-Picture support (iOS Safari 15+)
- Auto-hiding controls after 3 seconds of inactivity

**Phase 5: Performance Optimization**
- Skeleton screens during content loading
- Lazy loading for images as users scroll
- Cumulative Layout Shift prevention
- Offline state indicator
- 60fps smooth animations and transitions
- Fast response times across all interactions

### Key Features and Improvements

- **Adaptive Grid System**: Auto-fill grid that adjusts from 1 column (mobile) to 6+ columns (desktop)
- **Touch Target Sizing**: All interactive elements minimum 44x44px for iOS accessibility
- **Viewport Optimization**: Font size 16px on inputs to prevent auto-zoom
- **Responsive Typography**: Dynamic font sizes that scale with viewport width
- **Safe Area Support**: Proper padding for notched devices (iPhone X+)
- **Mobile-First CSS**: Base styles for mobile, enhanced for desktop with media queries
- **Cross-Device Navigation**: Automatic bottom nav (mobile) and top nav (desktop) switching
- **Accessibility**: Proper semantic HTML, ARIA labels, keyboard navigation support

### Browser and Device Compatibility

**Supported Browsers:**
- Safari 13+ (iOS)
- Chrome 90+ (Android)
- Firefox 88+ (Android)
- Samsung Internet 14+

**Supported Devices:**
- iPhone SE (320px) - baseline
- iPhone 12/13/14 (390px)
- iPhone 14 Pro (393px)
- iPhone Pro Max (430px)
- iPad (768px - 1024px)
- iPad Pro (1024px+)
- Android phones (various)
- Desktop browsers (1024px+)

**Known Limitations:**
- Internet Explorer 11 not supported (uses modern CSS Grid, Flexbox, CSS Custom Properties)
- Pre-iOS 13 not supported
- Android < 5.0 has limited support for CSS Grid

---

## 2. Testing Checklist

### Phase 1: Core Responsiveness

#### Viewport and Layout Testing

- [ ] Test on iPhone SE (320px width) - verify no horizontal scroll
- [ ] Test on iPhone 14 (390px width) - verify proper spacing
- [ ] Test on iPad (768px width) - verify grid columns increase
- [ ] Test on iPad Pro (1024px+) - verify maximum width constraints
- [ ] Test on desktop (1920px) - verify max-width container (1400px for media grid)
- [ ] Rotate device between portrait and landscape - layout reflows correctly

#### Touch Target Verification

- [ ] Verify all buttons have minimum 44x44px height
- [ ] Check buttons include padding, not just font size
- [ ] Test form input fields - tap and interact without difficulty
- [ ] Verify form group spacing provides clear button separation
- [ ] Test close buttons (X) on modals - are 44x44px circles
- [ ] Check tab buttons (Login/Register) - have sufficient height

#### Form Input Testing

- [ ] Type in text inputs on iOS - verify font size is 16px (no auto-zoom)
- [ ] Select from dropdowns on mobile - verify proper height
- [ ] Test password inputs - verify cursor visibility
- [ ] Test on iOS Safari with autofill - verify proper formatting
- [ ] Test on Android - verify keyboard doesn't overlay form
- [ ] Check form labels are visible and readable

#### Modal and Dialog Layout

- [ ] Open a modal on mobile (portrait) - verify width is 90%, max-width 500px
- [ ] Open modal on mobile (landscape) - verify still fits without scroll
- [ ] Test stacking of modal content on small screens
- [ ] Verify modal header and footer visible without scroll
- [ ] Check overflow content in modals has scroll capability
- [ ] Test modal backdrop opacity and dismiss on click

#### Grid and Layout Adaptation

- [ ] Media grid on 320px - verify shows 1-2 columns
- [ ] Media grid on 390px - verify shows 2 columns
- [ ] Media grid on 768px - verify shows 3-4 columns
- [ ] Media grid on 1024px - verify shows 4-5 columns
- [ ] Media grid on 1920px - verify shows 5-6 columns (max-width respected)
- [ ] Verify grid gap is consistent across all breakpoints

#### Safe Area Insets

- [ ] Test on iPhone X/11/12 (notch) in portrait - verify content doesn't overlap notch
- [ ] Test on iPhone X/11/12 in landscape - verify side safe areas respected
- [ ] Check bottom padding for home indicator on notched devices
- [ ] Test on iPad Pro with Dynamic Island (if available) - no overlap
- [ ] Verify nav bar adapts to safe areas

### Phase 2: Navigation

#### Bottom Navigation (Mobile <768px)

- [ ] Verify bottom nav appears only on screens <768px wide
- [ ] Test bottom nav on iPhone SE (320px) - proper icon and label visibility
- [ ] Test bottom nav on iPhone 14 (390px) - proper spacing
- [ ] Verify bottom nav height is appropriate (~60px)
- [ ] Test all nav items are tappable (44px+ touch targets)
- [ ] Check nav background color contrast against text

#### Navigation Drawer: Browse

- [ ] Tap "Browse" button - drawer slides in from left
- [ ] Verify drawer shows "Movies", "Shows", "Extras" options
- [ ] Each option is 44px+ height for touch
- [ ] Click "Movies" - navigates to Movies view and closes drawer
- [ ] Click "Shows" - navigates to Shows view and closes drawer
- [ ] Click "Extras" - navigates to Extras view and closes drawer
- [ ] Click backdrop - drawer closes
- [ ] Swipe down in drawer - drawer closes
- [ ] Verify drawer slides smoothly without lag

#### Navigation Drawer: More

- [ ] Tap "More" button - drawer slides in from left
- [ ] Verify drawer shows "Sections", "Admin", "Logout" options
- [ ] Each option is 44px+ height for touch
- [ ] Click "Sections" - shows sections admin (if available)
- [ ] Click "Admin" - navigates to admin panel
- [ ] Click "Logout" - logs out user and shows login screen
- [ ] Click backdrop - drawer closes
- [ ] Swipe down in drawer - drawer closes

#### Bottom Navigation State

- [ ] Navigate to Movies - "Movies" nav item shows active state (highlighted color)
- [ ] Navigate to Shows - "Shows" nav item now shows active state
- [ ] Navigate to Home - "Home" nav item shows active state
- [ ] Active state color should be distinct (typically blue #1e88e5)
- [ ] Previous active item returns to inactive state

#### Desktop Navigation (≥768px)

- [ ] Verify bottom nav is hidden on tablet/desktop
- [ ] Verify top navigation bar is visible instead
- [ ] Top nav has Browse and More dropdowns/menus
- [ ] Top nav layout doesn't conflict with content
- [ ] All navigation items remain accessible

#### Drawer Gestures

- [ ] Open drawer - verify it animates in smoothly
- [ ] Tap outside drawer (backdrop) - drawer closes smoothly
- [ ] Swipe down anywhere in drawer - drawer dismisses
- [ ] Multiple open/close cycles - no animation lag
- [ ] Verify drawer z-index is above page content

### Phase 3: Touch Interactions

#### Play Button Visibility

- [ ] View media grid on iOS - verify play buttons are always visible (not hover-only)
- [ ] View media grid on Android - verify play buttons are visible
- [ ] Tap play button - media starts playing
- [ ] Test on all device sizes - play buttons never hidden until interaction
- [ ] Verify play button is visually prominent (blue, 60px+ diameter)

#### Long-Press Playlist Activation

- [ ] Long-press (hold 500ms) on a playlist item - item becomes draggable
- [ ] Visual feedback shows item is selected for drag (opacity change or highlight)
- [ ] Short-press (<500ms) on playlist item - normal action (edit/open)
- [ ] Long-press on non-draggable item - no spurious activation
- [ ] Test timing with haptic feedback (if available)

#### Playlist Drag and Reorder

- [ ] After long-press activates drag, drag item up/down - it moves
- [ ] Verify drag handle indicator is visible
- [ ] Drop item in new position - order updates
- [ ] Drag item to top of list - moves to position 1
- [ ] Drag item to bottom of list - moves to last position
- [ ] Multiple drag operations - smooth reordering
- [ ] Swipe/drag over other items - they reorder dynamically

#### Haptic Feedback (iOS)

- [ ] (If supported) Long-press activation provides haptic feedback
- [ ] (If supported) Item reorder provides haptic on placement
- [ ] (If supported) Feedback is brief and non-intrusive
- [ ] Disable haptics in Settings - feedback stops

#### Swipe-Down Modal Dismiss

- [ ] Open a modal on mobile
- [ ] Swipe down from top of modal - modal starts to slide down
- [ ] Complete swipe - modal dismisses
- [ ] Partial swipe without completing - modal springs back
- [ ] Works on detail modals, player modal, any stacked modal
- [ ] Prevents accidental dismissal

#### Touch Feedback

- [ ] Tap any button - visual feedback (color change, scale, etc.)
- [ ] Tap any link - visual feedback
- [ ] Touch response is immediate (no lag)
- [ ] Visual feedback clears quickly after release
- [ ] Works on media cards, nav items, form buttons, etc.

### Phase 4: Enhanced Features

#### Pull-to-Refresh

- [ ] Navigate to Home view
- [ ] Pull down from top - refresh indicator appears
- [ ] Release pull - refresh animation starts
- [ ] Content refreshes and updates
- [ ] Same test on Movies view
- [ ] Same test on Shows view
- [ ] Same test on Extras view
- [ ] Verify pull-to-refresh doesn't conflict with scroll
- [ ] After refresh, content list updates (if changes exist)
- [ ] Refresh indicator has loading animation

#### Double-Tap Video Player - Rewind

- [ ] Start playing a video
- [ ] Double-tap left side of video - video jumps back 10 seconds
- [ ] Current time marker updates accordingly
- [ ] If at beginning, doesn't go negative
- [ ] Visual feedback shows rewind action (maybe "10s" indicator)
- [ ] Multiple double-taps work successively

#### Double-Tap Video Player - Forward

- [ ] Playing a video
- [ ] Double-tap right side of video - video jumps forward 10 seconds
- [ ] Current time marker updates
- [ ] If near end, doesn't exceed duration
- [ ] Visual feedback shows forward action
- [ ] Multiple double-taps work successively

#### Single-Tap Video Controls Toggle

- [ ] Video playing with controls visible
- [ ] Single tap on video - controls fade out (or hide)
- [ ] Controls auto-hide after 3 seconds of no interaction
- [ ] Single tap while hidden - controls show
- [ ] Tap outside controls while showing - controls remain
- [ ] Interact with control (scrub, volume, etc.) - controls stay visible

#### Swipe Volume Control

- [ ] Video player open
- [ ] Vertical swipe on right side of screen - volume indicator appears
- [ ] Swipe up - volume increases
- [ ] Swipe down - volume decreases
- [ ] Visual volume meter shows current level
- [ ] Works in portrait and landscape

#### Fullscreen Button

- [ ] Video player shows fullscreen button in controls
- [ ] Click fullscreen button - video expands to full screen
- [ ] Player controls adapt to fullscreen
- [ ] Click fullscreen again (or back button) - returns to normal
- [ ] Works on mobile, tablet, desktop

#### Picture-in-Picture Support

- [ ] (iOS Safari 15+) Video playing in player
- [ ] Click PiP button in player controls - video shrinks to corner
- [ ] Can resize PiP window
- [ ] Can move PiP window around screen
- [ ] Audio continues playing
- [ ] Navigate away - PiP continues
- [ ] Tap PiP window to expand back to full player

#### Auto-Hide Controls

- [ ] Video starts playing with controls visible
- [ ] After 3 seconds of no user input - controls fade out
- [ ] Move mouse/touch video - controls appear
- [ ] After another 3 seconds of inactivity - controls hide again
- [ ] Tap to show, wait 3 seconds, tap again to show - works consistently

### Phase 5: Performance

#### Skeleton Screens

- [ ] Navigate to Movies view - skeleton/loading placeholders appear
- [ ] Grid shows placeholder cards while content loads
- [ ] After content loads, placeholders are replaced smoothly
- [ ] Same behavior for Shows, Extras, Home
- [ ] Skeleton screens are visually distinct from actual content
- [ ] No visible pop-in or layout shift when content arrives

#### Image Lazy Loading

- [ ] Load Home view with many media cards
- [ ] Scroll down - images load as they come into viewport
- [ ] Images above fold are loaded first
- [ ] Images below fold load on-demand
- [ ] Verify HTTP requests show lazy loading pattern (not all at once)
- [ ] Verify scroll performance is smooth (60fps)

#### Layout Shift Prevention (CLS)

- [ ] Load a page with media cards
- [ ] As images load - cards don't shift position
- [ ] Card height is set before image loads (aspect-ratio CSS)
- [ ] No "pop-in" of images that moves other content
- [ ] Scroll smoothly without unexpected jumps
- [ ] Chrome DevTools - LCP and CLS metrics are good

#### Offline Indicator

- [ ] Disconnect network (airplane mode)
- [ ] Try to load page or refresh
- [ ] Offline indicator appears (banner or toast)
- [ ] Reconnect network
- [ ] Indicator disappears
- [ ] Can manually refresh and retry loading

#### Smooth Animations (60fps)

- [ ] Scroll media grid - smooth 60fps scrolling
- [ ] Open drawer - smooth slide animation
- [ ] Tap button - smooth color/scale transition
- [ ] Load image - smooth fade-in
- [ ] Use Chrome DevTools Performance tab - no frame drops
- [ ] On lower-end devices - animations still smooth (may be 30fps)

#### Fast Response Times

- [ ] Tap a button - immediate visual feedback (<100ms)
- [ ] Tap a link - page navigation starts immediately
- [ ] Pull to refresh - refresh indication appears immediately
- [ ] Type in search - filtering happens with <300ms delay
- [ ] Navigate between tabs - view switches within 500ms

---

## 3. iOS Wrapper App Testing

The iOS app wraps the web interface in a native WebView, providing native integration while using the responsive web design.

### Web Interface Integration

- [ ] Launch iOS app - web interface loads
- [ ] Verify app displays correctly in safe areas
- [ ] Home button takes you to home page
- [ ] Back button works (uses WebView back navigation)
- [ ] Forward button works
- [ ] Reload button refreshes current page

### JavaScript Bridge Communication

- [ ] Open Safari Web Inspector (plug into Mac) or check console logs
- [ ] Verify console shows "Media Server Native Bridge initialized"
- [ ] Check console for any JavaScript errors during load
- [ ] Navigate to different pages - no bridge errors in console

### Authentication Persistence

- [ ] Log in to web interface
- [ ] Close iOS app (swipe up to close)
- [ ] Reopen iOS app
- [ ] Verify you're still logged in (not redirected to login)
- [ ] Auth token persists across app launches
- [ ] Navigate app - token remains valid

### Deep Linking

- [ ] Share a movie/show link from web
- [ ] Copy the deep link URL (mediaserver://)
- [ ] Paste in Notes app
- [ ] Tap the link
- [ ] iOS app opens and navigates to correct content
- [ ] Verify correct media is displayed

### Share Functionality

- [ ] View a movie detail page
- [ ] Tap Share button (if available in web UI)
- [ ] iOS share sheet appears
- [ ] Select a destination (Messages, Mail, etc.)
- [ ] Content is properly shared
- [ ] Recipient receives correct information

### Background Audio

- [ ] Start playing video in app
- [ ] Press home button (app goes to background)
- [ ] Audio continues playing
- [ ] Lock device - audio continues
- [ ] Control Center shows playback controls
- [ ] Can pause/play from Control Center

### Safe Area Handling

- [ ] Launch app on iPhone X/11/12 (notch)
- [ ] Verify content doesn't overlap notch
- [ ] Bottom nav respects home indicator space
- [ ] Landscape orientation - side safe areas respected
- [ ] Modals don't overlap safe areas

### Pull-to-Refresh Conflict

- [ ] Pull-to-refresh active in web view
- [ ] Pull down slowly - web refresh indicator appears
- [ ] Pull down on scrollable content - web refresh works, doesn't trigger native refresh
- [ ] Release - web refresh completes
- [ ] Verify no double-refresh or conflicting behavior

### WebView Configuration

- [ ] JavaScript is enabled (needed for web interface)
- [ ] Media autoplay is allowed (for videos)
- [ ] Inline media playback is allowed
- [ ] Picture-in-Picture is allowed
- [ ] User agent includes "MediaServerNative"

---

## 4. Known Issues and Limitations

### Known Issues

**Issue: Hover Effects on Touch Devices**
- **Description**: Some pages may show hover states on touch devices when they shouldn't
- **Status**: Partially fixed - media cards show permanent play buttons, but some elements may have residual hover styles
- **Workaround**: Refresh page or navigate away and back
- **Fix**: All interactive elements should use `@media (hover: hover)` to disable hover on touch devices

**Issue: Pull-to-Refresh Timing**
- **Description**: Occasionally, pull-to-refresh may conflict with momentum scroll on iOS
- **Status**: Known limitation
- **Workaround**: Release pull gesture deliberately before refreshing
- **Fix**: Implement more sophisticated gesture detection

**Issue: Long-Press Context Menu on iOS**
- **Description**: Long-press on playlist items may trigger iOS context menu instead of drag
- **Status**: Works with proper 500ms timing, but device settings affect behavior
- **Workaround**: Use drag handle icon instead of full item area
- **Fix**: Use pointer-events and -webkit-touch-callout: none; CSS

**Issue: Video Player on Lower-End Android**
- **Description**: Video player may stutter on Android devices with <2GB RAM
- **Status**: Accepted limitation
- **Workaround**: Close other apps before playing videos
- **Impact**: Affects ~15% of Android devices

### Browser Compatibility Notes

- **Safari 13.1 (iOS)**: Full support, all features work
- **Chrome Android**: Full support with possible exception of Picture-in-Picture
- **Firefox Android**: Full support
- **Samsung Internet**: Full support (includes Samsung-specific optimizations)

### Device-Specific Limitations

**iPhone SE (1st Gen)**
- Limited to iOS 13 max
- Baseline for 320px testing
- Some older devices may have performance issues

**iPad Compatibility**
- Bottom nav may appear on larger iPads (768px)
- Recommend iPad Air or newer for smooth performance
- Older iPad models may have JavaScript performance limitations

**Android < 5.0**
- CSS Grid not fully supported
- Recommend Android 5.0+ for best experience
- ~5% of Android market share uses older versions

---

## 5. Deployment Instructions

### Web Interface Deployment

The web interface is served from the backend server. To deploy updates:

```bash
# 1. Backup existing file
cp /path/to/web/index.html /path/to/web/index.html.backup

# 2. Copy new index.html to web directory
# (Replace with your actual deployment command)
cp ./web/index.html /var/www/media-server/web/index.html

# 3. Restart the backend server to clear any caches
sudo systemctl restart media-server
# OR for Docker:
docker restart media-server-container

# 4. Verify deployment
curl http://localhost:8080 | head -20  # Check HTML loads
# Open http://localhost:8080 in browser and verify layout
```

**Deployment Checklist:**
- [ ] Backup current index.html
- [ ] Copy new index.html to production
- [ ] Clear browser cache (or use cache-busting query params)
- [ ] Restart server/clear server cache
- [ ] Test on mobile device (not desktop)
- [ ] Verify all assets load (CSS, images)
- [ ] Check responsive behavior on multiple devices
- [ ] Monitor server logs for errors

### iOS App Deployment

```bash
# 1. Navigate to iOS project directory
cd apps/ios-mobile

# 2. Update version number in Xcode
# Edit MediaServerMobile/Info.plist or project settings:
# - CFBundleShortVersionString (e.g., "1.0.0")
# - CFBundleVersion (build number, e.g., "42")

# 3. Build for device
xcodebuild -scheme MediaServerMobile -destination 'platform=iOS,name=Your Device'

# 4. Archive for distribution
xcodebuild -scheme MediaServerMobile -archivePath build/MediaServerMobile.xcarchive archive

# 5. Export for distribution (App Store, TestFlight, or Ad Hoc)
xcodebuild -exportArchive \
    -archivePath build/MediaServerMobile.xcarchive \
    -exportPath build/MediaServerMobile.ipa \
    -exportOptionsPlist ExportOptions.plist

# 6. Distribute via:
# - App Store Connect (TestFlight or Production)
# - Ad Hoc distribution (email .ipa)
# - Enterprise distribution
```

**iOS Build Checklist:**
- [ ] Update version number
- [ ] Update build number
- [ ] Verify signing certificate is valid
- [ ] Test build on real device (not simulator)
- [ ] Test on minimum iOS version (iOS 13)
- [ ] Test on latest iOS version
- [ ] Verify app permissions in Info.plist
- [ ] Check for console warnings/errors
- [ ] Test all deep links work
- [ ] Verify auth persists across app close/reopen
- [ ] Archive and test exported .ipa
- [ ] Upload to App Store Connect or distribute

### Deployment Verification Checklist

After deploying both web and app:

**Web Interface:**
- [ ] Load page on iPhone (test device)
- [ ] Verify responsive layout
- [ ] Test navigation (bottom nav on mobile, top nav on desktop)
- [ ] Test pull-to-refresh
- [ ] Test media playback
- [ ] Check console for errors
- [ ] Verify no horizontal scroll
- [ ] Check images load correctly
- [ ] Test on 2-3 different devices if possible

**iOS App:**
- [ ] Launch app on test iPhone
- [ ] Web interface loads
- [ ] Web content displays correctly
- [ ] Test navigation
- [ ] Verify safe areas respected
- [ ] Log in and verify persistence
- [ ] Test deep links (if applicable)
- [ ] Check device console (Xcode) for errors

---

## 6. Future Enhancements

### Potential Improvements for Future Iterations

**Performance**
- [ ] Service Worker for offline caching
- [ ] HTTP/2 Server Push for critical assets
- [ ] WebP image format support with JPEG fallback
- [ ] Image optimization on server (multiple sizes for breakpoints)
- [ ] CSS/JavaScript bundling and minification
- [ ] Brotli compression for text assets

**Features**
- [ ] Swipe gestures for prev/next media
- [ ] Customizable bottom nav items (user preferences)
- [ ] Dark/light theme toggle
- [ ] Customizable accent colors
- [ ] Advanced filtering in movie/show lists
- [ ] Smart search with suggestions
- [ ] User profiles (remember position per user)
- [ ] Watched/unwatched indicators
- [ ] Collections feature (custom playlists)

**Accessibility**
- [ ] Keyboard navigation improvements
- [ ] Screen reader optimization
- [ ] High contrast mode support
- [ ] Larger text option
- [ ] Closed captions support
- [ ] Audio descriptions

**iOS App**
- [ ] Widgets (upcoming video, continue watching)
- [ ] Siri Shortcuts integration
- [ ] Home app integration
- [ ] Universal Links instead of custom scheme
- [ ] iCloud sync for user preferences
- [ ] Handoff support (continue on other Apple devices)

**Android App**
- [ ] Native Android app (not just WebView wrapper)
- [ ] Android widgets
- [ ] Cast support (Chromecast)
- [ ] Media controls on lock screen
- [ ] Split screen support

**Cross-Platform**
- [ ] Web app manifest (installable PWA)
- [ ] Add to home screen support
- [ ] Offline playback caching
- [ ] Background sync
- [ ] Push notifications
- [ ] Real-time notifications

---

## 7. Troubleshooting Guide

### Common Issues and Solutions

#### Page Not Loading / Blank Screen

**Symptom**: App opens but shows blank white/black screen

**Solutions**:
1. Check internet connection - verify device has WiFi/cellular
2. Verify server is running:
   ```bash
   curl http://localhost:8080
   ```
3. Check server URL in app settings (iOS app)
4. Clear app cache:
   - iOS: Settings > Media Server > Clear Cache
   - Web: Clear browser cache (Ctrl+Shift+Delete)
5. Restart app/refresh page
6. Check server logs for errors

#### Content Not Loading / Spinner Forever

**Symptom**: Page loads but media cards never appear

**Solutions**:
1. Verify internet connection
2. Check server logs: `tail -f /var/log/media-server/server.log`
3. Verify API endpoint is accessible:
   ```bash
   curl http://localhost:8080/api/media
   ```
4. Refresh page (pull-to-refresh or reload button)
5. Check browser Network tab (Safari DevTools) for failed requests
6. Verify media content actually exists in server

#### Bottom Navigation Not Visible (Mobile)

**Symptom**: Bottom nav bar doesn't appear on mobile devices

**Solutions**:
1. Verify device width is <768px (check with device info)
2. Clear browser cache
3. Force responsive mode in browser DevTools (if testing on desktop)
4. Check browser zoom level (zoom should be 100%)
5. Verify CSS media queries are loading

#### Navigation Drawer Won't Open

**Symptom**: Tapping Browse/More button doesn't open drawer

**Solutions**:
1. Verify JavaScript is enabled
2. Check browser console for errors
3. Try force-closing and reopening app
4. Clear app cache
5. Verify touch coordinates by tapping multiple times
6. Check if drawer is hidden behind content (z-index issue)

#### Video Won't Play

**Symptom**: Video player opens but video doesn't play

**Solutions**:
1. Verify media file exists on server
2. Check browser Network tab - verify video file is loading
3. Verify browser supports video codec (check browser console)
4. Try different video (test with a known working video)
5. Check server logs for transcoding errors
6. Verify device has sufficient storage space
7. Try clearing app cache and reopening

#### Touch Targets Too Small

**Symptom**: Difficult to tap buttons, buttons are missing targets

**Solutions**:
1. This is a bug - report with device info (model, iOS/Android version)
2. Workaround: Use keyboard navigation if available
3. Check CSS includes 44px minimum (should not happen)
4. Verify browser zoom is 100%
5. Verify screen dimensions match expected device

#### Slow Performance / Stutter

**Symptom**: Scroll is laggy, animations stutter, page feels slow

**Solutions**:
1. Verify device has sufficient memory:
   - iOS: Check Settings > General > iPhone Storage
   - Android: Check Settings > Storage
2. Close other apps to free memory
3. Restart device
4. Disable animations in accessibility settings (iOS):
   - Settings > Accessibility > Display & Text Size > Reduce Motion
5. Reduce number of media items displayed (close some media cards)
6. Clear app cache
7. Update device to latest OS version
8. If server is on same device, verify server isn't consuming too much CPU

#### Images Not Loading

**Symptom**: Media cards show black/gray boxes instead of poster images

**Solutions**:
1. Verify internet connection
2. Check server logs for image processing errors:
   ```bash
   grep -i "image\|poster" /var/log/media-server/server.log
   ```
3. Verify image files exist on server
4. Verify server image caching is working
5. Force refresh page
6. Clear browser cache
7. Try accessing image URL directly in browser:
   ```
   http://localhost:8080/api/media/{mediaId}/poster
   ```

#### Responsive Layout Broken

**Symptom**: Layout looks wrong on mobile (elements overlapping, text too small, etc.)

**Solutions**:
1. Verify viewport meta tag is in HTML:
   ```html
   <meta name="viewport" content="width=device-width, initial-scale=1.0">
   ```
2. Verify CSS media queries are loading
3. Force reload page (Cmd+Shift+R or Ctrl+Shift+R)
4. Clear all browser caches
5. Disable browser extensions (especially ad blockers that modify CSS)
6. Test in private/incognito mode
7. Test on different browser (Safari vs Chrome)
8. Verify no custom zoom is applied

#### Authentication Issues

**Symptom**: Can't log in, logged out randomly, auth token expires too quickly

**Solutions**:
1. Verify username and password are correct
2. Check server time synchronization (auth tokens are time-based)
   ```bash
   date
   sudo timedatectl set-ntp true
   ```
3. Clear auth tokens and re-login:
   - iOS: Settings > Media Server > Clear Auth
   - Web: Clear site data in browser
4. Check server logs for auth errors:
   ```bash
   grep -i "auth\|token" /var/log/media-server/server.log
   ```
5. Verify SSL certificate is valid (if using HTTPS)
6. Check if JWT token expiration is set correctly in server config

#### Safe Area Issues (Notched Devices)

**Symptom**: Content overlaps notch on iPhone X/11/12, or bottom nav overlaps home indicator

**Solutions**:
1. This is a bug - should not happen
2. Force reload page
3. Restart app
4. Verify Safe Area CSS is applied:
   ```css
   padding-top: max(12px, env(safe-area-inset-top));
   padding-bottom: max(12px, env(safe-area-inset-bottom));
   ```
5. Check iOS version (must be iOS 13+)
6. Verify device is in correct orientation

#### Pull-to-Refresh Not Working

**Symptom**: Pull-to-refresh doesn't trigger refresh

**Solutions**:
1. Verify you're at top of scrollable content
2. Pull down firmly (not too light, not too fast)
3. Wait for refresh indicator to appear
4. Release to trigger refresh
5. If still not working, try soft refresh instead (button)
6. Check browser console for JavaScript errors

#### Deep Link Not Working (iOS App)

**Symptom**: Tapping a mediaserver:// URL doesn't open app or open correct content

**Solutions**:
1. Verify URL format is correct: `mediaserver://media/{id}`
2. Verify mediaserver:// scheme is registered in Info.plist
3. Restart iOS app
4. Try reopening from fresh state (not backgrounded)
5. Check if URL contains special characters (may need encoding)
6. Verify media ID actually exists
7. Check Xcode console for deep link handling errors

#### Push Notifications Not Working

**Symptom**: Don't receive notifications from server

**Solutions**:
1. Verify notifications are enabled in app settings
2. Verify notifications are enabled in iOS Settings:
   - Settings > Notifications > Media Server > Allow Notifications
3. Check if iPhone is in Do Not Disturb mode
4. Force-kill and reopen app to reset notification permission
5. Check server logs for notification delivery errors
6. Verify certificate is valid for push notifications

---

## Appendix A: Browser DevTools Reference

### Safari Web Inspector (iOS)

To debug web content in iOS app:

1. Connect iPhone to Mac
2. Open Safari on Mac
3. Menu: Develop > [Your iPhone] > Media Server Mobile
4. Web Inspector opens, showing:
   - HTML Inspector
   - Console (JavaScript logs and errors)
   - Network tab (HTTP requests)
   - Storage (localStorage, cookies)

### Chrome DevTools (Android)

To debug Android WebView:

1. Connect Android device with USB
2. Open Chrome on desktop
3. Visit chrome://inspect
4. Enable "Discover USB devices"
5. Open Media Server app on Android
6. Device should appear in Chrome DevTools
7. Click "Inspect" to open DevTools

### Responsive Design Mode (All Browsers)

To test responsive design on desktop:

**Safari:**
- Menu: Develop > Enter Responsive Design Mode

**Chrome:**
- Menu: More tools > Responsive Design Mode
- Or press Ctrl+Shift+M (Cmd+Shift+M on Mac)

**Firefox:**
- Menu: Tools > Browser Tools > Responsive Design Mode
- Or press Ctrl+Shift+M

---

## Appendix B: CSS Breakpoints Reference

The responsive design uses these breakpoints:

```css
/* Mobile: 320px - 767px */
@media (max-width: 767px) {
    /* 1-2 column grids */
    /* Bottom navigation visible */
    /* Compact spacing */
}

/* Tablet: 768px - 1023px */
@media (min-width: 768px) and (max-width: 1023px) {
    /* 3-4 column grids */
    /* No bottom navigation */
    /* Standard spacing */
}

/* Desktop: 1024px+ */
@media (min-width: 1024px) {
    /* 4-6 column grids */
    /* Max-width constraints (1400px) */
    /* Full spacing */
}

/* Large Desktop: 1920px+ */
@media (min-width: 1920px) {
    /* 6+ column grids */
    /* Maximum content width enforced */
}
```

### Custom Properties (CSS Variables)

```css
:root {
    /* Spacing */
    --container-padding-mobile: 12px;
    --container-padding-desktop: 20px;
    --touch-target-min: 44px;

    /* Colors */
    --primary-blue: #1e88e5;
    --dark-bg: #1a1a2e;
    --card-bg: #16213e;

    /* Typography */
    --font-stack: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}
```

---

## Appendix C: Testing Device Specifications

### Reference Device Screen Sizes

| Device | Width | Height | DPI | Aspect |
|--------|-------|--------|-----|--------|
| iPhone SE (1st) | 320px | 568px | 163ppi | 16:9 |
| iPhone 6/7/8 | 375px | 667px | 163ppi | 16:9 |
| iPhone 12 mini | 375px | 812px | 476ppi | 19.5:9 |
| iPhone 12/13 | 390px | 844px | 458ppi | 19.5:9 |
| iPhone 14 | 390px | 844px | 460ppi | 19.5:9 |
| iPhone 14 Plus | 428px | 926px | 460ppi | 19.5:9 |
| iPhone 14 Pro | 393px | 852px | 460ppi | 19.5:9 |
| iPhone 14 Pro Max | 430px | 932px | 460ppi | 19.5:9 |
| iPhone X/11 Pro | 375px | 812px | 458ppi | 19.5:9 |
| iPad (7th gen) | 768px | 1024px | 163ppi | 4:3 |
| iPad Air | 768px | 1024px | 264ppi | 4:3 |
| iPad Pro 11" | 834px | 1194px | 264ppi | ~17.5:12 |
| iPad Pro 12.9" | 1024px | 1366px | 264ppi | ~4:3 |
| Android (Galaxy S21) | 360px | 800px | 420ppi | 18:9 |
| Android (Pixel 6) | 412px | 915px | 420ppi | 19.5:9 |

---

## Appendix D: Performance Targets

### Lighthouse Scoring Targets

- **Performance**: 90+
- **Accessibility**: 90+
- **Best Practices**: 90+
- **SEO**: 90+

### Core Web Vitals

- **Largest Contentful Paint (LCP)**: < 2.5 seconds
- **First Input Delay (FID)**: < 100 milliseconds
- **Cumulative Layout Shift (CLS)**: < 0.1

### Other Performance Metrics

- **Time to First Byte (TTFB)**: < 600ms
- **First Contentful Paint (FCP)**: < 1.8 seconds
- **Fully Interactive (TTI)**: < 3.8 seconds
- **Total Page Size**: < 2MB
- **Initial JS Bundle**: < 500KB

---

## Appendix E: Support and Escalation

### For Issues Not Covered in This Guide

1. **Check the troubleshooting section** (Section 7)
2. **Check browser console** for error messages
3. **Check server logs**:
   ```bash
   tail -f /var/log/media-server/server.log
   ```
4. **Try on different device** (to isolate device-specific issue)
5. **Try on different browser** (to isolate browser-specific issue)
6. **Report with these details**:
   - Device model (iPhone 14, Galaxy S21, etc.)
   - OS version (iOS 16.2, Android 12, etc.)
   - Browser (Safari, Chrome, Firefox, etc.)
   - Steps to reproduce
   - Screenshots or screen recording
   - Server logs
   - Browser console errors

---

## Document Version History

- **v1.0** (2025-01-26): Initial comprehensive guide for mobile web redesign
  - Covers all 5 phases of implementation
  - Includes detailed testing checklists
  - Documents iOS wrapper app integration
  - Provides deployment instructions
  - Includes troubleshooting guide

---

**Last Updated**: January 26, 2025
**Maintained By**: Development Team
**Status**: Active
