@implementation Good

@property(weak) Foo* foo;
@property(weak, readonly) id<A> foo;
@property(atomic, weak) Foo* foo;
@property(strong) id<Foo> foo;
@property(readwrite, strong) Foo* foo;
@property(strong, atomic) Foo* foo;
@property(copy) id<Foo> foo;
@property(copy, readwrite) Foo* foo;
@property(copy, atomic) id<Foo> foo;
@property(assign) Foo* foo;
@property(atomic, assign) id<Foo> foo;
@property(assign, nonatomic) Foo* foo;
@property BOOL foo;
@property(nonatomic) int foo;
@property(atomic) BOOL foo;

@end
