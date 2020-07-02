



### Worker pools and outgoing connections
            
Allocate new worker pool with a count of workers that can handle a second of maximum incoming requests limit-i
Maximum size of pool can be calculated as urls-count / limit-c * limit-i * limit-c = 20 / 4 * 100 * 4 = 2000.
It's number of all workers that should be allocated to cover the worst case scenario for the long run: 100 incoming requests with 20 urls and
each url takes little less a second for each second.

```
=========================== SECONDS =================================>
1(4)  2(4)  3(4)  4(4)  5(4)
400 - 400 - 400 - 400 - 400 - releasing workers
      1(4)  2(4)  3(4)  4(4)  5(4)
      400 - 400 - 400 - 400 - 400 - releasing workers
            1(4)  2(4)  3(4)  4(4)  5(4)
            400 - 400 - 400 - 400 - 400 - releasing workers
                  1(4)  2(4)  3(4)  4(4)  5(4)
                  400 - 400 - 400 - 400 - 400 - releasing workers
                        1(4)  2(4)  3(4)  4(4)  5(4)
                        400 - 400 - 400 - 400 - 400 - releasing workers
                         ↑    1(4)  2(4)  3(4)  4(4)  5(4)
                 max preasure 400 - 400 - 400 - 400 - 400 - releasing workers
.......................................................................
```

This is pretty artificial scenario, so elastic pool grow chosen to avoid useless resource consumptions.
