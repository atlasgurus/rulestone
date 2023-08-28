package com.atlasgurus.rulestone.util;

import org.newsclub.net.unix.AFUNIXSocket;
import org.newsclub.net.unix.AFUNIXSocketAddress;

import java.io.BufferedOutputStream;
import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.nio.file.Files;
import java.nio.file.Path;

public class SidecarProcess {

  private Process goSidecarProcess;
  public InputStream inputStream;
  public OutputStream outputStream;
  private final int SEND_BUFFER_SIZE = 4 * 1024 * 1024;
  private final int RECV_BUFFER_SIZE = 4 * 1024 * 1024;

  private final int OUTPUT_BUFFER_SIZE = 8 * 1024;

  private static Path sidecarBinaryPath = null;

  static {
    try {
      extractSidecarBinary();
    } catch (IOException e) {
      throw new RuntimeException(e);
    } catch (InterruptedException e) {
      throw new RuntimeException(e);
    }
  }

  private static String getSidecarBinaryName() {
    String osName = System.getProperty("os.name").toLowerCase();
    String osArch = System.getProperty("os.arch");

    String binaryName = "rulestone_sidecar";

    if (osName.contains("mac")) {
      binaryName += "_darwin";
    } else if (osName.contains("linux")) {
      binaryName += "_linux";
    } else {
      throw new RuntimeException("Unsupported OS: " + osName);
    }

    if (osArch.contains("x86_64") || osArch.contains("amd64")) {
      binaryName += "_amd64";
    } else if (osArch.contains("arm64")) {
      binaryName += "_arm64";
    } else {
      throw new RuntimeException("Unsupported architecture: " + osArch);
    }

    return binaryName;
  }

  private static void extractSidecarBinary() throws IOException, InterruptedException {
    if (sidecarBinaryPath == null) {
      sidecarBinaryPath = Files.createTempFile(getSidecarBinaryName() + "_", null);

      try (InputStream is = SidecarProcess.class.getResourceAsStream("/" + getSidecarBinaryName());
          OutputStream os = Files.newOutputStream(sidecarBinaryPath)) {
        byte[] buffer = new byte[1024];
        int bytesRead;

        while ((bytesRead = is.read(buffer)) != -1) {
          os.write(buffer, 0, bytesRead);
        }
      }

      // 2. Make the binary executable (For Unix-like systems)
      if (!System.getProperty("os.name").toLowerCase().contains("win")) {
        ProcessBuilder chmod = new ProcessBuilder("chmod", "+x", sidecarBinaryPath.toString());
        chmod.inheritIO().start().waitFor();
      }
    }
  }

  public SidecarProcess() throws IOException {
    ProcessBuilder processBuilder = new ProcessBuilder(sidecarBinaryPath.toString());
    processBuilder.redirectError(new File("sidecar_error.log"));
    processBuilder.redirectOutput(new File("sidecar_output.log"));
    goSidecarProcess = processBuilder.start();

    File socketFile = new File("/tmp/go_sidecar.sock");
    AFUNIXSocketAddress address = new AFUNIXSocketAddress(socketFile);

    AFUNIXSocket socket = AFUNIXSocket.newInstance();

    // Set the send buffer size
    socket.setSendBufferSize(SEND_BUFFER_SIZE);

    // Set the receive buffer size
    socket.setReceiveBufferSize(RECV_BUFFER_SIZE);

    socket.connect(address);

    inputStream = socket.getInputStream();
    outputStream = new BufferedOutputStream(socket.getOutputStream(), OUTPUT_BUFFER_SIZE);

    /* For some reason this gets triggered before the process exits
     */
    Runtime.getRuntime().addShutdownHook(new Thread(() -> {
      try {
        close();
      } catch (IOException e) {
        throw new RuntimeException(e);
      }
    }));
  }

  public void close() throws IOException {
    if (inputStream != null) {
      inputStream.close();
    }
    if (outputStream != null) {
      outputStream.close();
    }
    if (goSidecarProcess != null) {
      goSidecarProcess.destroy();
    }
  }
}
