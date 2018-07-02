package statefulservice;

import java.time.Duration;
import java.util.logging.Logger;
import java.util.logging.Level;

import microsoft.servicefabric.services.runtime.ServiceRuntime;

public class JavaServiceHost {

    private static final Logger logger = Logger.getLogger(JavaServiceHost.class.getName());

    public static void main(String[] args) throws Exception {
        try {
            ServiceRuntime.registerStatefulServiceAsync("JavaServiceType", (context) -> new JavaService(context), Duration.ofSeconds(10));
            logger.log(Level.INFO, "Registered stateful service of type JavaServiceType. ");
            Thread.sleep(Long.MAX_VALUE);
        } catch (Exception ex) {
            logger.log(Level.SEVERE, "Exception occured", ex);
            throw ex;
        }
    }
}
