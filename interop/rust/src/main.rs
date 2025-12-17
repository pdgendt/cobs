use bytes::BytesMut;
use clap::{Parser, Subcommand};
use cobs_codec::{Decoder, Encoder};
use std::io::{self, Read, Write};
use tokio_util::codec::Decoder as DecoderTrait;
use tokio_util::codec::Encoder as EncoderTrait;

// SENTINEL value can be set at compile time via environment variable:
// SENTINEL=10 cargo build
const SENTINEL: u8 = match env!("SENTINEL_VALUE").as_bytes() {
    [b] => *b - b'0',
    [b1, b2] => (*b1 - b'0') * 10 + (*b2 - b'0'),
    [b1, b2, b3] => (*b1 - b'0') * 100 + (*b2 - b'0') * 10 + (*b3 - b'0'),
    _ => 0,
};

#[derive(Parser)]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    Decode,
    Encode,
}

fn main() {
    let cli = Cli::parse();

    let mut input = Vec::new();
    io::stdin().read_to_end(&mut input).unwrap();

    match &cli.command {
        Commands::Decode => {
            let mut decoder = Decoder::<SENTINEL>::new();
            let mut src = BytesMut::from(&input[..]);
            let mut output = BytesMut::new();
            while let Some(frame) = decoder.decode(&mut src).unwrap() {
                output.extend_from_slice(&frame);
            }
            io::stdout().write_all(&output).unwrap();
        }
        Commands::Encode => {
            let mut encoder = Encoder::<SENTINEL>::new();
            let mut dst = BytesMut::new();
            encoder.encode(input, &mut dst).unwrap();
            io::stdout().write_all(&dst).unwrap();
        }
    }

    io::stdout().flush().unwrap();
}
