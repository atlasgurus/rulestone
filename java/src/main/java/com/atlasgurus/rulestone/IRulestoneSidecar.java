package com.atlasgurus.rulestone;

import java.io.IOException;

public interface IRulestoneSidecar {

  int getRuleEngineId();

  int addRuleFromString(String ruleString) throws IOException;

  int addRuleFromFile(String rulePath) throws IOException;

  void addRulesFromDirectory(String rulesPath) throws IOException;

  void activate() throws IOException;

  public interface Callback {

    void onCompletion(int[] matches);
  }

  void sendRequest(String request, Callback cb) throws IOException;

  void close() throws InterruptedException, IOException;
}
