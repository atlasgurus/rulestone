package com.atlasgurus.rulestone;

import java.util.HashMap;
import java.util.Map;

public class RuleMetadata {

  private Map<String, Object> metadata = new HashMap();

  public RuleMetadata(Map<String, Object> metadata) {
    this.metadata = metadata;
  }

  public String getValue(String key) {
    return (String) this.metadata.get(key);
  }
}
