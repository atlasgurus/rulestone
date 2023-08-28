package com.atlasgurus.rulestone;

import java.io.IOException;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.BlockingQueue;

public class RulestoneSidecarProducerConsumer implements IRulestoneSidecar {

  // Define a concurrent queue to communicate between producer and consumer
  private static final int QUEUE_CAPACITY = 1000; // Adjust this to your desired size
  private final BlockingQueue<Object> requestQueue = new ArrayBlockingQueue<>(QUEUE_CAPACITY);

  private final Object END_OF_JOB_SENTINEL = new Object();

  int engineId = 0;

  private final int TIMEOUT_MS = 60 * 1000; // set timeout to 1min

  private final SidecarOps ops = new SidecarOps();
  private Thread consumer;

  public RulestoneSidecarProducerConsumer() throws IOException {}

  @Override
  public int addRuleFromString(String ruleString) throws IOException {
    return ops.addRuleFromString(ruleString);
  }

  @Override
  public int addRuleFromFile(String rulePath) throws IOException {
    return ops.addRuleFromFile(rulePath);
  }

  @Override
  public void addRulesFromDirectory(String rulesPath) throws IOException {
    ops.addRulesFromDirectory(rulesPath);
  }

  @Override
  public void activate() throws IOException {
    ops.activate();
    consumer = startConsumer();
  }

  // Method to send a request to Go and notify the consumer
  @Override
  public void sendRequest(String request, Callback cb) throws IOException {
    ops.matchRequest(request);

    // Add a marker (or the request itself) to the queue
    try {
      if (!requestQueue.offer(cb)) {
        ops.flush();
        requestQueue.put(cb);
      }
    } catch (InterruptedException e) {
      throw new RuntimeException(e);
    }
  }


  @Override
  public void close() throws InterruptedException, IOException {
    requestQueue.put(END_OF_JOB_SENTINEL);

    ops.flush();

    stopConsumer(consumer);
    ops.close();
  }

  public int getRuleEngineId() {
    return ops.getRuleEngineId();
  }

  private void stopConsumer(Thread consumer) throws InterruptedException {
    // Wait for the consumer thread to finish, but no longer than 5 minutes
    try {
      consumer.join(TIMEOUT_MS);
    } catch (InterruptedException e) {
      Thread.currentThread().interrupt();
      System.out
          .println("Main thread was interrupted while waiting for consumer thread to finish.");
    }

    if (consumer.isAlive()) {
      System.out.println(
          String.format("Consumer thread did not finish within the %d sec timeout period.",
              TIMEOUT_MS / 1000));
      consumer.interrupt(); // Interrupt the consumer thread
    }
  }

  private Thread startConsumer() {
    // Start the consumer thread
    Thread consumer = new Thread(() -> {
      try {
        while (true) {
          // Wait for a marker from the queue
          Object o = requestQueue.take();
          if (o == END_OF_JOB_SENTINEL) {
            break;
          }
          Callback cb = (Callback) o;
          cb.onCompletion(ops.matchResponse());
        }
      } catch (InterruptedException | IOException e) {
        Thread.currentThread().interrupt();
      }
    });
    consumer.start();
    return consumer;
  }
}
