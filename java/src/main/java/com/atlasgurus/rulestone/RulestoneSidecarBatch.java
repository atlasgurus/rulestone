package com.atlasgurus.rulestone;

import java.io.IOException;
import java.util.ArrayList;
import java.util.List;

public class RulestoneSidecarBatch implements IRulestoneSidecar {

  private static final int DEFAULT_BATCH_SIZE = 1000;

  private List<Callback> callbacks;

  private final SidecarOps sidecarOps = new SidecarOps();;

  private final int batchSize;

  public RulestoneSidecarBatch() throws IOException {
    this(DEFAULT_BATCH_SIZE);
  }

  public RulestoneSidecarBatch(int batchSize) throws IOException {
    this.batchSize = batchSize;
    this.callbacks = new ArrayList<>(DEFAULT_BATCH_SIZE);
  }

  public int getRuleEngineId() {
    return sidecarOps.getRuleEngineId();
  }

  @Override
  public int addRuleFromString(String ruleString) throws IOException {
    return sidecarOps.addRuleFromString(ruleString);
  }

  @Override
  public int addRuleFromFile(String rulePath) throws IOException {
    return sidecarOps.addRuleFromFile(rulePath);
  }

  @Override
  public void addRulesFromDirectory(String rulesDirectory) throws IOException {
    sidecarOps.addRulesFromDirectory(rulesDirectory);
  }

  @Override
  public void activate() throws IOException {
    sidecarOps.activate();
  }

  @Override
  public void sendRequest(String request, Callback cb) throws IOException {
    sidecarOps.matchRequest(request);
    callbacks.add(cb);
    if (callbacks.size() == this.batchSize) {
      flushBatch();
    }
  }

  private void flushBatch() throws IOException {
    sidecarOps.flush();
    if (!callbacks.isEmpty()) {
      try {
        for (Callback cb : callbacks) {
          cb.onCompletion(sidecarOps.matchResponse());
        }
      } catch (IOException e) {
        throw new RuntimeException(e);
      }
      // Process results if needed
      callbacks.clear();
    }
  }

  @Override
  public void close() throws IOException {
    flushBatch();
    // sidecarOps.close();
  }
}
