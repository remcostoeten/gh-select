//! Custom error types for gh-select

pub const GhSelectError = error{
    // GitHub API Errors
    GhCliNotInstalled,
    GhNotAuthenticated,
    GhApiFailed,
    GhJsonParseError,
    
    // Cache Errors
    CacheCreateFailed,
    CacheWriteFailed,
    CacheReadFailed,
    CacheExpired,
    
    // Clipboard Errors
    ClipboardUnavailable,
    ClipboardWriteFailed,
    
    // UI Errors
    SelectionCancelled,
    UserAborted,
};
