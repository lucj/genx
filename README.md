Simple data generator

Cosinus curve with the following caracteristics:
- duration of 2 days
- sampling done every 3 hour
- mininal value is 20
- maximal value is 30

```
$ docker run lucj/genx -type cos -duration 2d -min 20 -max 30 -step 3h
1538766162 26.35
1538776962 29.36
1538787762 29.81
1538798562 27.45
1538809362 23.65
1538820162 20.64
1538830962 20.19
1538841762 22.55
1538852562 26.35
1538863362 29.36
1538874162 29.81
1538884962 27.45
1538895762 23.65
1538906562 20.64
1538917362 20.19
1538928162 22.55
```

![Cosinus graph](/images/cos.png)

Linear curve with the following caracteristics:
- duration of 3 days
- sampling done every 6 hours
- first value is 10
- last value is 30


```
$ docker run lucj/genx -type linear -duration 3d -first 10 -last 30 -step 6h
1538766338 10.00
1538787938 11.67
1538809538 13.33
1538831138 15.00
1538852738 16.67
1538874338 18.33
1538895938 20.00
1538917538 21.67
1538939138 23.33
1538960738 25.00
1538982338 26.67
1539003938 28.33
```

![Linear graph](/images/linear.png)
