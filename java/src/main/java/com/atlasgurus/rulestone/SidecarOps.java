package com.atlasgurus.rulestone;

import com.atlasgurus.rulestone.util.Comm;
import com.atlasgurus.rulestone.util.SidecarProcess;

import java.io.IOException;


public class SidecarOps {

  private static final int COMMAND_CREATE_RULE_ENGINE = 1;

  private static final int COMMAND_ADD_RULE_FROM_JSON_STRING = 2;
  private static final int COMMAND_ADD_RULE_FROM_YAML_STRING = 3;
  private static final int COMMAND_ADD_RULE_FROM_FILE = 4;
  private static final int COMMAND_ADD_RULE_FROM_DIRECTORY = 5;
  private static final int COMMAND_ACTIVATE = 6;
  private static final int COMMAND_MATCH = 7;
  private static SidecarProcess sidecarProcess;

  static {
    try {
      sidecarProcess = new SidecarProcess();
    } catch (IOException e) {
      throw new RuntimeException(e);
    }
  }

  private int nextRequestId = 0;
  private int nextResponseId = 0;
  private final int engineId;

  public SidecarOps() throws IOException {
    // Write out the request
    Comm.writeInt16(sidecarProcess.outputStream, COMMAND_CREATE_RULE_ENGINE);
    flush();
    engineId = Comm.readInt16(sidecarProcess.inputStream);

    System.out.println("Initialized rule engine with ID: " + engineId);
  }

  public int addRuleFromString(String ruleString) throws IOException {
    // Write out the request
    Comm.writeInt16(sidecarProcess.outputStream, COMMAND_ADD_RULE_FROM_JSON_STRING);
    Comm.writeInt16(sidecarProcess.outputStream, engineId);
    Comm.writeLengthPrefixedMessage(sidecarProcess.outputStream, ruleString);

    flush();

    // Read the response
    int ruleId = Comm.readInt32(sidecarProcess.inputStream);
    if (ruleId < 0) {
      System.err.println(String.format("Failed to add rule %s", ruleString));
      throw new RuntimeException("Failed to initialize rule engine");
    }
    // System.out.println("Added rule ID: " + ruleId);
    return ruleId;
  }

  public int addRuleFromFile(String rulePath) throws IOException {
    // Write out the request
    Comm.writeInt16(sidecarProcess.outputStream, COMMAND_ADD_RULE_FROM_FILE);
    Comm.writeInt16(sidecarProcess.outputStream, engineId);
    Comm.writeLengthPrefixedMessage(sidecarProcess.outputStream, rulePath);

    flush();

    // Read the response
    int ruleId = Comm.readInt32(sidecarProcess.inputStream);
    if (ruleId < 0) {
      System.err.println(String.format("Failed to add rule from %s", rulePath));
      throw new RuntimeException("Failed to initialize rule engine");
    }
    // System.out.println("Added rule ID: " + ruleId);
    return ruleId;
  }

  public int addRulesFromDirectory(String rulePath) throws IOException {
    // Write out the request
    Comm.writeInt16(sidecarProcess.outputStream, COMMAND_ADD_RULE_FROM_DIRECTORY);
    Comm.writeInt16(sidecarProcess.outputStream, engineId);
    Comm.writeLengthPrefixedMessage(sidecarProcess.outputStream, rulePath);

    flush();

    // Read the response
    int numRules = Comm.readInt32(sidecarProcess.inputStream);
    if (numRules < 0) {
      System.err.println(String.format("Failed to add rules from %s", rulePath));
      throw new RuntimeException("Failed to initialize rule engine");
    }
    System.out.println(String.format("Added %d rules: ", numRules));
    return numRules;
  }

  public int getRuleEngineId() {
    return engineId;
  }


  public void activate() throws IOException {
    // Write out the request
    Comm.writeInt16(sidecarProcess.outputStream, COMMAND_ACTIVATE);
    Comm.writeInt16(sidecarProcess.outputStream, engineId);
    flush();
  }

  public int matchRequest(String jsonString) throws IOException {
    // Write out the request
    Comm.writeInt16(sidecarProcess.outputStream, COMMAND_MATCH);
    Comm.writeInt16(sidecarProcess.outputStream, engineId);
    // Comm.writeInt32(sidecarProcess.outputStream, ++nextRequestId);
    Comm.writeLengthPrefixedMessage(sidecarProcess.outputStream, jsonString);
    return nextRequestId;
  }

  public int[] matchResponse() throws IOException {
    // Read (blocking) response
    /*
     * int responseId = Comm.readInt32(sidecarProcess.inputStream); if (responseId !=
     * ++nextResponseId) { throw new
     * RuntimeException("Request/response sequence numbers do not match"); }
     */
    return Comm.readMatchList(sidecarProcess.inputStream);
  }

  public void close() throws IOException {
    sidecarProcess.close();
  }

  public void flush() throws IOException {
    sidecarProcess.outputStream.flush();
  }
}
