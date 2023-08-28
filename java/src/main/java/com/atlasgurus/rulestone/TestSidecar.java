package com.atlasgurus.rulestone;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.stream.Stream;

import static com.atlasgurus.rulestone.RulestoneSidecarFactory.*;

public class TestSidecar {

  public static void main(String[] args) throws InterruptedException, IOException {
    String directoryPath = "/users/vvg/rulestone/gen.configs.rulestone";
    String filePath = "/users/vvg/rulestone/rule_benchmark_data.jsonl";
    if (args.length > 1) {
      directoryPath = args[0];
      filePath = args[1];
    }


    //IRulestoneSidecar rs = getSidecarInstance(PRODUCER_CONSUMER_SIDECAR);
    IRulestoneSidecar rs = getSidecarInstance(BATCHING_SIDECAR);
    rs.addRulesFromDirectory(directoryPath);
    rs.activate();

    IRulestoneSidecar.Callback cb = new IRulestoneSidecar.Callback() {
      @Override
      public void onCompletion(int[] matches) {
      }
    };

    long startTime = System.nanoTime();
    try (Stream<String> lines = Files.lines(Paths.get(filePath))) {
      lines.forEach(line -> {
        try {
          rs.sendRequest(line, cb);
        } catch (IOException e) {
          throw new RuntimeException(e);
        }
      });
    }
    long endTime = System.nanoTime();
    long duration = (endTime - startTime);
    System.out.println("Producer/Consumer Execution time: " + duration / 1000000 + " ms");
    rs.close();
  }
}
