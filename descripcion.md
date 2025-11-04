# Calculadora de Reservas

We work in actuarial sciences and we want to make a calculator of reserves 

## Objetivo

Make a calculator of reserves with can be paralellized


## Explanation

We wish to calculate Reservas de Rentas Vitalicias given by the CMF instruction indications
CMF the regulator of finantial sector in Chile

The reserves depend highly on the tablas de mortalidad, which are given by the regulator to us.

## Stack Technology

We would like to use python for maintanability, but if we're making a product maybe we should also try using go,
we have to investigate.

I wish to use sqlite to store tablas de mortalidad, and to store polizas info 

Also i would like that we do the calculation then save the result by poliza in a new table of results.

There are some polizas that need to be calculated with different kinds of condition we have to investigate that

If we use python, it should be deployed as a conda env reservas

If we go for go; we should target Windows; Linux and Macos 


## Parallel

Every poliza which is defined, it has its unique conditions, this unique conditions makes so that we can in a way calculate reserves independent one for another.

The process can use the power of the multithreaded stuff to calculate fast
