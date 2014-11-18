package main

import "io"
import "bytes"
import "encoding/binary"


// MPEG version enum.
const (
    MpegVersion2_5 = iota
    MpegVersionReserved
    MpegVersion2
    MpegVersion1
)


// MPEG layer enum.
const (
    MpegLayerReserved = iota
    MpegLayerIII
    MpegLayerII
    MpegLayerI
)


// Channel mode enum.
const (
    Stereo = iota
    JointStereo
    DualChannel
    Mono
)


// Bit rates.
var v1l1_br = []int{0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448}
var v1l2_br = []int{0, 32, 48, 56,  64,  80,  96, 112, 128, 160, 192, 224, 256, 320, 384}
var v1l3_br = []int{0, 32, 40, 48,  56,  64,  80,  96, 112, 128, 160, 192, 224, 256, 320}
var v2l1_br = []int{0, 32, 48, 56,  64,  80,  96, 112, 128, 144, 160, 176, 192, 224, 256}
var v2l2_br = []int{0,  8, 16, 24,  32,  40,  48,  56,  64,  80,  96, 112, 128, 144, 160}
var v2l3_br = []int{0,  8, 16, 24,  32,  40,  48,  56,  64,  80,  96, 112, 128, 144, 160}


// Sampling rates.
var v1_sr  = []int{44100, 48000, 32000}
var v2_sr  = []int{22050, 24000, 16000}
var v25_sr = []int{11025, 12000,  8000}


// Mp3Frame represents a single frame parsed from an MP3 stream.
type Mp3Frame struct {
    MpegVersion byte
    MpegLayer byte
    CrcProtection bool
    BitRate int
    SamplingRate int
    PaddingBit bool
    PrivateBit bool
    ChannelMode byte
    ModeExtension byte
    CopyrightBit bool
    OriginalBit bool
    Emphasis byte
    SampleCount int
    FrameLength int
    RawBytes []byte
}


// nextFrame loads the next MP3 frame from the input stream.
// Returns nil when the stream has been exhausted.
func nextFrame(stream io.Reader) *Mp3Frame {

    // Each MP3 frame begins with a 4-byte header.
    buffer := make([]byte, 4)
    lastByte := buffer[3:]
    frame := &Mp3Frame{}

    // Fill the header buffer.
    if ok := fillBuffer(stream, buffer); !ok {
        return nil
    }

    // Loop until we find a frame or reach the end of the stream.
    for {

        // Check for an ID3v1 tag: 'TAG'.
        if buffer[0] == 84 && buffer[1] == 65 && buffer[2] == 71 {

            debug("skipping ID3v1 tag")

            // Read the remainder of the tag and discard it.
            // ID3v1 tags have a fixed length of 128 bytes.
            remainder := make([]byte, 124)
            if ok := fillBuffer(stream, remainder); !ok {
                return nil
            }

            // Refill the header buffer.
            if ok := fillBuffer(stream, buffer); !ok {
                return nil
            }
        }

        // Check for an ID3v2 tag: 'ID3'.
        if buffer[0] == 73 && buffer[1] == 68 && buffer[2] == 51 {

            debug("skipping ID3v2 tag")

            // Read the remainder of the 10 byte tag header.
            remainder := make([]byte, 6)
            if ok := fillBuffer(stream, remainder); !ok {
                return nil
            }

            // The last 4 bytes of the header indicate the length of the tag.
            // This length does not include the header itself.
            length :=
                (int(remainder[2]) << (7 * 3)) |
                (int(remainder[3]) << (7 * 2)) |
                (int(remainder[4]) << (7 * 1)) |
                (int(remainder[5]) << (7 * 0))

            // Read the tag data and discard it.
            data := make([]byte, length)
            if ok := fillBuffer(stream, data); !ok {
                return nil
            }

            // Refill the header buffer.
            if ok := fillBuffer(stream, buffer); !ok {
                return nil
            }
        }

        // Check for a frame header, indicated by an 11-bit frame-sync sequence.
        if buffer[0] == 0xFF && (buffer[1] & 0xE0) == 0xE0 {

            // Check if the header is valid. If it is, parse its values
            // into our Mp3Frame struct.
            if ok := parseHeader(buffer, frame); ok {

                frame.RawBytes = make([]byte, frame.FrameLength)
                copy(frame.RawBytes, buffer)

                if ok := fillBuffer(stream, frame.RawBytes[4:]); !ok {
                    return nil
                }

                return frame
            }
        }

        // No frame yet. Shift the buffer forward by one byte and try again.
        debug("sync error: skipping byte")
        buffer[0] = buffer[1]
        buffer[1] = buffer[2]
        buffer[2] = buffer[3]
        n, _ := stream.Read(lastByte)
        if n < 1 {
            return nil
        }
    }
}


// parseHeader attempts to parse a slice of 4 bytes as a valid MP3 header.
// The return value is a boolean indicating success. If the header is valid
// its values are written into the supplied Mp3Frame struct.
func parseHeader(header []byte, frame *Mp3Frame) bool {

    // MPEG version. (2 bits)
    frame.MpegVersion = (header[1] & 0x18) >> 3
    if frame.MpegVersion == MpegVersionReserved {
        return false
    }

    // MPEG layer. (2 bits.)
    frame.MpegLayer = (header[1] & 0x06) >> 1
    if frame.MpegLayer == MpegLayerReserved {
        return false
    }

    // CRC (cyclic redundency check) protection. (1 bit.)
    frame.CrcProtection = (header[1] & 0x01) == 0x00

    // Bit rate index. (4 bits.)
    bitRateIndex := (header[2] & 0xF0) >> 4
    if bitRateIndex == 0 || bitRateIndex == 15 {
        return false
    }

    // Bit rate.
    if frame.MpegVersion == MpegVersion1 {
        switch frame.MpegLayer {
        case MpegLayerI:   frame.BitRate = v1l1_br[bitRateIndex] * 1000
        case MpegLayerII:  frame.BitRate = v1l2_br[bitRateIndex] * 1000
        case MpegLayerIII: frame.BitRate = v1l3_br[bitRateIndex] * 1000
        }
    } else {
        switch frame.MpegLayer {
        case MpegLayerI:   frame.BitRate = v2l1_br[bitRateIndex] * 1000
        case MpegLayerII:  frame.BitRate = v2l2_br[bitRateIndex] * 1000
        case MpegLayerIII: frame.BitRate = v2l3_br[bitRateIndex] * 1000
        }
    }

    // Sampling rate index. (2 bits.)
    samplingRateIndex := (header[2] & 0x0C) >> 2
    if samplingRateIndex == 3 {
        return false
    }

    // Sampling rate.
    switch frame.MpegVersion {
    case MpegVersion1:   frame.SamplingRate = v1_sr[samplingRateIndex]
    case MpegVersion2:   frame.SamplingRate = v2_sr[samplingRateIndex]
    case MpegVersion2_5: frame.SamplingRate = v25_sr[samplingRateIndex]
    }

    // Padding bit. (1 bit.)
    frame.PaddingBit = (header[2] & 0x02) == 0x02

    // Private bit. (1 bit.)
    frame.PrivateBit = (header[2] & 0x01) == 0x01

    // Channel mode. (2 bits.)
    frame.ChannelMode = (header[3] & 0xC0) >> 6

    // Mode Extension. Valid only for Joint Stereo mode. (2 bits.)
    frame.ModeExtension = (header[3] & 0x30) >> 4
    if frame.ChannelMode != JointStereo && frame.ModeExtension != 0 {
        return false
    }

    // Copyright bit. (1 bit.)
    frame.CopyrightBit = (header[3] & 0x08) == 0x08

    // Original bit. (1 bit.)
    frame.OriginalBit = (header[3] & 0x04) == 0x04

    // Emphasis. (2 bits.)
    frame.Emphasis = (header[3] & 0x03)
    if frame.Emphasis == 2 {
        return false
    }

    // Number of samples in the frame. We need this to determine the frame size.
    if frame.MpegVersion == MpegVersion1 {
        switch frame.MpegLayer {
        case MpegLayerI:   frame.SampleCount = 384
        case MpegLayerII:  frame.SampleCount = 1152
        case MpegLayerIII: frame.SampleCount = 1152
        }
    } else {
        switch frame.MpegLayer {
        case MpegLayerI:   frame.SampleCount = 384
        case MpegLayerII:  frame.SampleCount = 1152
        case MpegLayerIII: frame.SampleCount = 576
        }
    }

    // If the padding bit is set we add an extra 'slot' to the frame length.
    // A layer I slot is 4 bytes long; layer II and III slots are 1 byte long.
    var padding int = 0

    if frame.PaddingBit {
        if frame.MpegLayer == MpegLayerI {
            padding = 4
        } else {
            padding = 1
        }
    }

    // Calculate the frame length in bytes.
    // There's a lot of confusion online about how to do this and definitive
    // documentation is hard to find. The basic formula seems to boil down to:
    //
    //     bytes_per_sample = (bit_rate / sampling_rate) / 8
    //     frame_length = sample_count * bytes_per_sample + padding
    //
    // In practice we need to rearrange this formula to avoid rounding errors.
    //
    // I can't find any definitive statement on whether this length is supposed
    // to include the 4-byte header and the optional 2-byte CRC. Experimentation
    // on mp3 files captured from the wild indicates that it includes the header
    // at least.
    frame.FrameLength =
        (frame.SampleCount / 8) * frame.BitRate / frame.SamplingRate + padding

    return true
}


func getSideInformationSize(frame *Mp3Frame) (size int) {

    if frame.MpegLayer == MpegLayerIII {
        if frame.MpegVersion == MpegVersion1 {
            if frame.ChannelMode == Mono {
                size = 17
            } else {
                size = 32
            }
        } else {
            if frame.ChannelMode == Mono {
                size = 9
            } else {
                size = 17
            }
        }
    }

    return size
}


func isXingHeader(frame *Mp3Frame) bool {

    // 4 bytes for the frame header plus the size of the side information block.
    offset := 4 + getSideInformationSize(frame)
    id := frame.RawBytes[offset:offset + 4]

    if bytes.Equal(id, []byte("Xing")) || bytes.Equal(id, []byte("Info")) {
        return true
    }

    return false
}


func isVbriHeader(frame *Mp3Frame) bool {

    // 4 bytes for the frame header plus a fixed 32-byte offset.
    id := frame.RawBytes[36:36 + 4]

    if bytes.Equal(id, []byte("VBRI")) {
        return true
    }

    return false
}


func newXingHeader(template *Mp3Frame, totalFrames, totalBytes uint32) *Mp3Frame {

    // Make a shallow copy of the template frame.
    xingFrame := *template

    // Make a new zeroed-out data slice.
    xingFrame.RawBytes = make([]byte, xingFrame.FrameLength)

    // Copy over the frame header.
    copy(xingFrame.RawBytes, template.RawBytes[:4])

    // Determine the Xing header offset.
    offset := 4 + getSideInformationSize(template)

    // Write the Xing header ID.
    copy(xingFrame.RawBytes[offset:offset + 4], []byte("Xing"))

    // Write a flag indicating that the number-of-frames
    // and number-of-bytes fields are present.
    xingFrame.RawBytes[offset + 7] = 3

    // Write the number of frames as a 32-bit big endian integer.
    binary.BigEndian.PutUint32(xingFrame.RawBytes[offset + 8:offset + 12], totalFrames)

    // Write the number of bytes as a 32-bit big endian integer.
    binary.BigEndian.PutUint32(xingFrame.RawBytes[offset + 12:offset + 16], totalBytes)

    return &xingFrame
}
