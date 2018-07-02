package statefulservice;

import java.time.Duration;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.CompletableFuture;
import java.util.logging.Level;
import java.util.logging.Logger;

import microsoft.servicefabric.data.ReliableStateManager;
import microsoft.servicefabric.data.Transaction;
import microsoft.servicefabric.services.communication.runtime.ServiceReplicaListener;
import microsoft.servicefabric.services.runtime.StatefulService;
import microsoft.servicefabric.data.collections.ReliableHashMap;
import system.fabric.CancellationToken;
import system.fabric.StatefulServiceContext;

public class JavaService extends StatefulService {
    private ReliableStateManager stateManager;
    private static final Logger logger = Logger.getLogger(JavaService.class.getName());
    
    protected JavaService (StatefulServiceContext statefulServiceContext) {
        super (statefulServiceContext);
        this.stateManager = this.getReliableStateManager();
    }

    @Override
    protected List<ServiceReplicaListener> createServiceReplicaListeners() {
        // Create your own ServiceReplicaListeners and add to the listenerList.
        List<ServiceReplicaListener> listenerList = new ArrayList<>();
        // listenerList.add(listener1);
        return listenerList;
    }

    @Override
    protected CompletableFuture<?> runAsync(CancellationToken cancellationToken) {
        // TODO: Replace the following sample code with your own logic
        // or remove this runAsync override if it's not needed in your service.

        Transaction tx = stateManager.createTransaction();
        CompletableFuture<ReliableHashMap<String, Long>> mapFuture = this.stateManager
                .<String, Long>getOrAddReliableHashMapAsync("myHashMap");

        return mapFuture.thenCompose((map) -> {
            return computeValueAsync(map, tx, cancellationToken);
        }).thenCompose((v) -> {
            return commitTransaction(tx);
        }).whenComplete((res, e) -> {
            closeTransaction(tx);
        });
    }

    private static CompletableFuture<Long> computeValueAsync(ReliableHashMap<String, Long> map, Transaction tx,
            CancellationToken token) {
        return map.computeAsync(tx, "counter", (k, v) -> {
            if (v == null)
                return 1L;
            else
                return ++v;
        }, Duration.ofSeconds(4), token);
    }

    private static CompletableFuture<?> commitTransaction(Transaction tx) {
        return tx.commitAsync();
    }

    private static void closeTransaction(Transaction tx) {
        try {
            tx.close();
        } catch (Exception e) {
            logger.log(Level.SEVERE, "Exception in closing transaction", e);
        }
    }
}

