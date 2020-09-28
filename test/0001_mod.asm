mod: SUBI SP,SP,#16   // reserve space for n,LR
    STUR LR,[SP,#8]  // 
    STUR X0,[SP,#0]  //
    SUBS X0,X1,X0    //
    B.GT L1          //
    CBZ X0,L2        //
    BL mod           //
    B L2             //
L:  LDUR X0,[SP,#0]  //
L2: LDUR LR,[SP,#8]  // 
    ADDI SP,SP,#16
    BR LR