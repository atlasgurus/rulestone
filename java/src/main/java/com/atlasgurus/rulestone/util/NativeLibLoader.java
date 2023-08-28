package com.atlasgurus.rulestone.util;

import java.io.File;
import java.io.FileNotFoundException;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;

public class NativeLibLoader {

  public static void loadLibrary(String name) {
    try {
      System.load(extractLibrary(name));
    } catch (IOException e) {
      e.printStackTrace();
    }
  }

  private static String extractLibrary(String name) throws IOException {
    String osName = System.getProperty("os.name").toLowerCase();
    String osArch = System.getProperty("os.arch");

    String libExtension = osName.contains("mac") ? ".dylib" : ".so";

    String libFullName = name + "_" + osArch + libExtension;
    InputStream in = null;
    OutputStream out = null;

    try {
      in = NativeLibLoader.class.getResourceAsStream("/" + libFullName);
      if (in == null) {
        throw new FileNotFoundException(libFullName);
      }

      File fileOut = File.createTempFile(name, libExtension);
      out = new FileOutputStream(fileOut);

      byte[] buffer = new byte[1024];
      int read;

      while ((read = in.read(buffer)) != -1) {
        out.write(buffer, 0, read);
      }

      return fileOut.getAbsolutePath();
    } finally {
      if (in != null) {
        in.close();
      }
      if (out != null) {
        out.close();
      }
    }
  }

  public static void main(String[] args) {
    System.out.println(System.getProperty("os.arch"));
  }
}
