package com.atlasgurus.rulestone;

import java.io.IOException;

public class RulestoneSidecarFactory {

  public static final int BATCHING_SIDECAR = 1;
  public static final int PRODUCER_CONSUMER_SIDECAR = 2;

  public static IRulestoneSidecar getSidecarInstance(int type) {
    switch (type) {
      case BATCHING_SIDECAR:
        // Assuming BatchingSidecar is a class that implements IRulestoneSidecar
        try {
          return new RulestoneSidecarBatch();
        } catch (IOException e) {
          throw new RuntimeException(e);
        }
      case PRODUCER_CONSUMER_SIDECAR:
        // Assuming ProducerConsumerSidecar is a class that implements IRulestoneSidecar
        try {
          return new RulestoneSidecarProducerConsumer();
        } catch (IOException e) {
          throw new RuntimeException(e);
        }
      default:
        throw new IllegalArgumentException("Invalid type provided: " + type);
    }
  }
}
