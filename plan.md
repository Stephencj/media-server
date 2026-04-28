# Universal Remote Channel Button Support for Apple TV

## Problem
Universal remotes with dedicated channel up/down buttons send `UIPress.PressType.pageUp` and `UIPress.PressType.pageDown` events. SwiftUI's `.onMoveCommand` only handles D-pad directions, not these dedicated buttons.

## Solution
Create a UIKit-based press handler wrapped in `UIViewRepresentable` to capture pageUp/pageDown events and trigger channel switching.

## Files to Create

### 1. `Components/RemoteButtonHandler.swift`
A SwiftUI-compatible wrapper that captures UIPress events for pageUp/pageDown buttons.

```swift
import SwiftUI
import UIKit

struct RemoteButtonHandler: UIViewRepresentable {
    let onPageUp: () -> Void
    let onPageDown: () -> Void

    func makeUIView(context: Context) -> RemoteButtonView {
        let view = RemoteButtonView()
        view.onPageUp = onPageUp
        view.onPageDown = onPageDown
        return view
    }

    func updateUIView(_ uiView: RemoteButtonView, context: Context) {
        uiView.onPageUp = onPageUp
        uiView.onPageDown = onPageDown
    }
}

class RemoteButtonView: UIView {
    var onPageUp: (() -> Void)?
    var onPageDown: (() -> Void)?

    override var canBecomeFocused: Bool { true }

    override func pressesBegan(_ presses: Set<UIPress>, with event: UIPressesEvent?) {
        for press in presses {
            switch press.type {
            case .pageUp:
                onPageUp?()
                return
            case .pageDown:
                onPageDown?()
                return
            default:
                break
            }
        }
        super.pressesBegan(presses, with: event)
    }
}
```

## Files to Modify

### 1. `Views/ChannelPlayerView.swift`
Add the RemoteButtonHandler as a background layer to capture page button events.

**Changes:**
- Import the RemoteButtonHandler
- Add it as a background view that captures pageUp/pageDown
- Wire callbacks to existing `switchToNextChannel()` and `switchToPreviousChannel()` functions

```swift
// In the body, add as background:
.background(
    RemoteButtonHandler(
        onPageUp: { switchToNextChannel() },
        onPageDown: { switchToPreviousChannel() }
    )
)
```

### 2. `MediaPlayer.xcodeproj/project.pbxproj`
Add the new RemoteButtonHandler.swift file to the project.

## Implementation Notes

1. **Focus Handling**: The RemoteButtonView needs `canBecomeFocused = true` to receive press events. However, since it's a background view, we may need to use gesture recognizers instead for better focus handling.

2. **Alternative Approach**: If the background view approach has focus issues, use `UITapGestureRecognizer` with `allowedPressTypes`:
   ```swift
   let pageUpGesture = UITapGestureRecognizer(target: self, action: #selector(handlePageUp))
   pageUpGesture.allowedPressTypes = [NSNumber(value: UIPress.PressType.pageUp.rawValue)]
   ```

3. **Testing**: Requires a universal remote with channel buttons to test. The Apple TV simulator doesn't support pageUp/pageDown.

## Verification

1. Build the app successfully
2. Test with Siri Remote - D-pad up/down should still work via `.onMoveCommand`
3. Test with universal remote - Channel Up/Down buttons should trigger channel switching
4. Both input methods should show the channel switch overlay
