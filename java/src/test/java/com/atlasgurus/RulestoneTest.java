package com.atlasgurus;

import com.atlasgurus.rulestone.*;
import org.junit.Test;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.concurrent.CountDownLatch;

import static org.junit.Assert.*;


public class RulestoneTest {

  @Test
  public void testSingleRuleBatch() throws IOException, InterruptedException {
    IRulestoneSidecar sidecar = new RulestoneSidecarBatch(1);
    testRules(sidecar, "./src/test/resources/rules/single", "./src/test/resources/data/data.json",
        new int[] {0});
  }

  @Test
  public void testSingleRuleProducerConsumer() throws IOException, InterruptedException {
    IRulestoneSidecar sidecar = new RulestoneSidecarProducerConsumer();
    testRules(sidecar, "./src/test/resources/rules/single", "./src/test/resources/data/data.json",
        new int[] {0});
  }

  @Test
  public void testMultipleRulesBatch() throws IOException, InterruptedException {
    IRulestoneSidecar sidecar = new RulestoneSidecarBatch(1);
    testRules(sidecar, "./src/test/resources/rules/multiple",
        "./src/test/resources/data/data.json", new int[] {1, 0});
  }

  @Test
  public void testMultipleRulesProducerConsumer() throws IOException, InterruptedException {
    IRulestoneSidecar sidecar = new RulestoneSidecarBatch(1);
    testRules(sidecar, "./src/test/resources/rules/multiple",
        "./src/test/resources/data/data.json", new int[] {1, 0});
  }

  public void testRules(IRulestoneSidecar sidecar, String rulesDir, String dataPath, int[] expected)
      throws IOException, InterruptedException {
    sidecar.addRulesFromDirectory(rulesDir);

    String testInput =
        new String(Files.readAllBytes(Paths.get(dataPath)));

    sidecar.activate();
    sidecar.sendRequest(testInput, new IRulestoneSidecar.Callback() {
      @Override
      public void onCompletion(int[] matches) {
        assertEquals(expected.length, matches.length);
        assertArrayEquals(expected, matches);
      }
    });
  }
}
