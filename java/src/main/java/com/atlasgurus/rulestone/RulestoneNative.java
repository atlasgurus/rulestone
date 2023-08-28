package com.atlasgurus.rulestone;

import com.atlasgurus.rulestone.util.NativeLibLoader;

import java.util.HashMap;
import java.util.Map;

public class RulestoneNative {

  static {
    NativeLibLoader.loadLibrary("librulestone");
  }

  private static native int NewRuleEngine(String rule_path);

  private static native int[] Match(int ruleEngineId, String input_json);

  private static native RuleMetadata GetRuleMetadata(int ruleEngineId, int ruleId);

  private int ruleEngineId;

  private static Map<String, RulestoneNative> instanceMap = new HashMap<>();
  private static Map<Integer, RuleMetadata> metadataCache = new HashMap<>();

  public static RulestoneNative getInstance(String rulePath) {
    RulestoneNative instance = null;
    if (!instanceMap.containsKey(rulePath)) {
      instance = new RulestoneNative(rulePath);
      instanceMap.put(rulePath, instance);
    } else {
      instance = instanceMap.get(rulePath);
    }
    return instance;
  }

  private RulestoneNative(String rulePath) {
    ruleEngineId = NewRuleEngine(rulePath);
  }

  public int[] match(String input_json) {
    return Match(this.ruleEngineId, input_json);
  }

  public RuleMetadata getRuleMetadata(int ruleId) {
    if (metadataCache.containsKey(ruleId)) {
      return metadataCache.get(ruleId);
    } else {
      RuleMetadata metadata = GetRuleMetadata(this.ruleEngineId, ruleId);
      metadataCache.put(ruleId, metadata);
      return metadata;
    }
  }
}
