package com.atlasgurus.rulestone.util;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.nio.IntBuffer;

public class Comm {

  public static void writeInt16(OutputStream os, int v) throws IOException {
    byte[] buf = {
        (byte) ((v >> 8) & 0xFF),
        (byte) (v & 0xFF)};

    os.write(buf);
  }

  public static void writeInt32(OutputStream os, int v) throws IOException {
    byte[] buf = {
        (byte) ((v >> 24) & 0xFF),
        (byte) ((v >> 16) & 0xFF),
        (byte) ((v >> 8) & 0xFF),
        (byte) (v & 0xFF)};

    os.write(buf);
  }

  public static void writeLengthPrefixedMessage(OutputStream os, String message)
      throws IOException {
    byte[] data = message.getBytes();
    writeInt32(os, data.length);
    os.write(data);
  }

  public static int readInt16(InputStream is) throws IOException {
    byte[] buf16 = new byte[2];
    readFully(is, buf16);
    return ((buf16[0] & 0xFF) << 8)
        | (buf16[1] & 0xFF);
  }

  public static int readInt32(InputStream is) throws IOException {
    byte[] buf32 = new byte[4];
    readFully(is, buf32);
    return ((buf32[0] & 0xFF) << 24)
        | ((buf32[1] & 0xFF) << 16)
        | ((buf32[2] & 0xFF) << 8)
        | (buf32[3] & 0xFF);
  }

  public static void readFully(InputStream input, byte[] buffer) throws IOException {
    int totalRead = input.read(buffer, 0, buffer.length);
    if (totalRead == -1) {
      throw new IOException("Unexpected end of stream");
    }
    while (totalRead < buffer.length) {
      int bytesRead = input.read(buffer, totalRead, buffer.length - totalRead);
      if (bytesRead == -1) {
        throw new IOException("Unexpected end of stream");
      }
      totalRead += bytesRead;
    }
  }

  public static int[] readMatchList(InputStream is) throws IOException {
    int length = Comm.readInt32(is);

    byte[] byteBuff = new byte[length * 4];
    ByteBuffer buffer = ByteBuffer.allocate(length * 4).order(ByteOrder.LITTLE_ENDIAN); // <-- Set
    // to
    // little-endian

    Comm.readFully(is, byteBuff);
    buffer.put(byteBuff);
    buffer.flip();
    IntBuffer intBuffer = buffer.asIntBuffer();
    int[] results = new int[intBuffer.remaining()];
    intBuffer.get(results);
    return results;
  }
}
