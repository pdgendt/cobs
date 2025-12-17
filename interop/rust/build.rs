use std::env;

fn main() {
    // Read SENTINEL environment variable, default to 0 if not set
    let sentinel = env::var("SENTINEL")
        .ok()
        .and_then(|s| s.parse::<u8>().ok())
        .unwrap_or(0);

    // Make the value available to the code via cfg
    println!("cargo:rustc-env=SENTINEL_VALUE={}", sentinel);

    // Rerun build script if SENTINEL env var changes
    println!("cargo:rerun-if-env-changed=SENTINEL");
}
