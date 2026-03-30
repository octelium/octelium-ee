import { Condition } from "@/apis/corev1/corev1";
import { SegmentedControl } from "@mantine/core";
import * as React from "react";
import { match } from "ts-pattern";
import EditItem from "../EditItem";
import ItemMessage from "../ItemMessage";
import { CELEditor, OPAEditor } from "./Editor";

import ConditionBuilderBtn from "../PolicyBuilder/ConditionBuilderBtn";

const Cond = (props: {
  item?: Condition;
  onChange: (condition: Condition) => void;
}) => {
  const { item } = props;
  let [req, setReq] = React.useState(
    props.item ??
      Condition.create({
        type: {
          oneofKind: "match",
          match: "",
        },
      }),
  );

  let [init, _] = React.useState(Condition.clone(req));

  const updateReq = () => {
    setReq(Condition.clone(req));
    props.onChange(req);
  };

  React.useEffect(() => {
    setReq(
      item ??
        Condition.create({
          type: {
            oneofKind: `match`,
            match: ``,
          },
        }),
    );
  }, [item]);

  return (
    <div className="ml-3 w-full my-4">
      <div className="flex flex-col">
        <SegmentedControl
          defaultValue={req.type.oneofKind}
          classNames={{
            label: "!font-bold !transition-all !duration-500",
          }}
          data={[
            {
              value: "match",
              label: "Match",
            },
            {
              value: "not",
              label: "Match Not",
            },
            {
              value: "all",
              label: "All of (AND)",
            },
            {
              value: "any",
              label: "Any of (OR)",
            },
            {
              value: "none",
              label: "None of (NOR)",
            },
            {
              value: "opa",
              label: "OPA",
            },
            {
              value: "matchAny",
              label: "Match Everything",
            },
          ]}
          onChange={(v) => {
            match(v)
              .with("match", () => {
                match(init.type)
                  .when(
                    (x) => x.oneofKind === `match`,
                    (x) => {
                      req.type = x;
                    },
                  )
                  .otherwise(() => {
                    req = Condition.create({
                      type: {
                        oneofKind: "match",
                        match: "",
                      },
                    });
                  });

                updateReq();
              })
              .with("not", () => {
                match(init.type)
                  .when(
                    (x) => x.oneofKind === `not`,
                    (x) => {
                      req.type = x;
                    },
                  )
                  .otherwise(() => {
                    req = Condition.create({
                      type: {
                        oneofKind: "not",
                        not: "",
                      },
                    });
                  });

                updateReq();
              })
              .with("all", () => {
                match(init.type)
                  .when(
                    (x) => x.oneofKind === `all`,
                    (x) => {
                      req.type = x;
                    },
                  )
                  .otherwise(() => {
                    req = Condition.create({
                      type: {
                        oneofKind: "all",
                        all: {
                          of: [
                            Condition.create({
                              type: {
                                oneofKind: "match",
                                match: "",
                              },
                            }),
                          ],
                        },
                      },
                    });
                  });

                updateReq();
              })
              .with("any", () => {
                match(init.type)
                  .when(
                    (x) => x.oneofKind === `any`,
                    (x) => {
                      req.type = x;
                    },
                  )
                  .otherwise(() => {
                    req = Condition.create({
                      type: {
                        oneofKind: "any",
                        any: {
                          of: [
                            Condition.create({
                              type: {
                                oneofKind: "match",
                                match: "",
                              },
                            }),
                          ],
                        },
                      },
                    });
                  });

                updateReq();
              })
              .with("none", () => {
                match(init.type)
                  .when(
                    (x) => x.oneofKind === `none`,
                    (x) => {
                      req.type = x;
                    },
                  )
                  .otherwise(() => {
                    req = Condition.create({
                      type: {
                        oneofKind: "none",
                        none: {
                          of: [
                            Condition.create({
                              type: {
                                oneofKind: "match",
                                match: "",
                              },
                            }),
                          ],
                        },
                      },
                    });
                  });

                updateReq();
              })
              .with("opa", () => {
                match(init.type)
                  .when(
                    (x) => x.oneofKind === `opa`,
                    (x) => {
                      req.type = x;
                    },
                  )
                  .otherwise(() => {
                    req = Condition.create({
                      type: {
                        oneofKind: "opa",
                        opa: {
                          type: {
                            oneofKind: "inline",
                            inline: "",
                          },
                        },
                      },
                    });
                  });

                updateReq();
              })
              .with("matchAny", () => {
                match(init.type)
                  .when(
                    (x) => x.oneofKind === `matchAny`,
                    (x) => {
                      req.type = x;
                    },
                  )
                  .otherwise(() => {
                    req = Condition.create({
                      type: {
                        oneofKind: "matchAny",
                        matchAny: true,
                      },
                    });
                  });

                updateReq();
              });
          }}
        />

        <div className="my-2 flex items-center">
          <ConditionBuilderBtn
            onChange={(v) => {
              req = v ?? Condition.create();
              updateReq();
            }}
          />
        </div>
      </div>
      {match(req.type)
        .when(
          (x) => x.oneofKind === `all`,
          (all) => {
            return (
              <ItemMessage
                title="All Conditions"
                obj={all.all.of}
                isList
                onSet={() => {
                  all.all.of = [
                    Condition.create({
                      type: {
                        oneofKind: "match",
                        match: "",
                      },
                    }),
                  ];
                  updateReq();
                }}
                onAddListItem={() => {
                  all.all.of.push(
                    Condition.create({
                      type: {
                        oneofKind: "match",
                        match: "",
                      },
                    }),
                  );

                  updateReq();
                }}
              >
                {all.all.of.map((x, idx) => (
                  <EditItem
                    obj={all.all.of.at(idx)}
                    key={`${idx}`}
                    onUnset={() => {
                      updateReq();
                    }}
                  >
                    <Cond
                      item={x}
                      onChange={(v) => {
                        all.all.of[idx] = v!;
                        updateReq();
                      }}
                    />
                  </EditItem>
                ))}
              </ItemMessage>
            );
          },
        )
        .when(
          (x) => x.oneofKind === `any`,
          (any) => {
            return (
              <ItemMessage
                title="Any Conditions"
                obj={any.any.of}
                isList
                onSet={() => {
                  any.any.of = [
                    Condition.create({
                      type: {
                        oneofKind: "match",
                        match: "",
                      },
                    }),
                  ];
                  updateReq();
                }}
                onAddListItem={() => {
                  any.any.of.push(
                    Condition.create({
                      type: {
                        oneofKind: "match",
                        match: "",
                      },
                    }),
                  );

                  updateReq();
                }}
              >
                {any.any.of.map((x, idx) => (
                  <EditItem
                    obj={any.any.of.at(idx)}
                    key={`${idx}`}
                    onUnset={() => {
                      updateReq();
                    }}
                  >
                    <Cond
                      item={x}
                      onChange={(v) => {
                        any.any.of[idx] = v!;
                        updateReq();
                      }}
                    />
                  </EditItem>
                ))}
              </ItemMessage>
            );
          },
        )
        .when(
          (x) => x.oneofKind === `none`,
          (none) => {
            return (
              <ItemMessage
                title="None Conditions"
                obj={none.none.of}
                isList
                onSet={() => {
                  none.none.of = [
                    Condition.create({
                      type: {
                        oneofKind: "match",
                        match: "",
                      },
                    }),
                  ];
                  updateReq();
                }}
                onAddListItem={() => {
                  none.none.of.push(
                    Condition.create({
                      type: {
                        oneofKind: "match",
                        match: "",
                      },
                    }),
                  );

                  updateReq();
                }}
              >
                {none.none.of.map((x, idx) => (
                  <EditItem
                    obj={none.none.of.at(idx)}
                    key={`${idx}`}
                    onUnset={() => {
                      updateReq();
                    }}
                  >
                    <Cond
                      item={x}
                      onChange={(v) => {
                        none.none.of[idx] = v!;
                        updateReq();
                      }}
                    />
                  </EditItem>
                ))}
              </ItemMessage>
            );
          },
        )
        .when(
          (x) => x.oneofKind === `match`,
          (match) => {
            return (
              <CELEditor
                exp={match.match}
                onChange={(v) => {
                  match.match = v;
                  updateReq();
                }}
              />
            );
          },
        )
        .when(
          (x) => x.oneofKind === `not`,
          (not) => {
            return (
              <CELEditor
                exp={not.not}
                onChange={(v) => {
                  not.not = v;

                  updateReq();
                }}
              />
            );
          },
        )
        .when(
          (x) => x.oneofKind === `opa`,
          (opa) => {
            return match(opa.opa.type)
              .when(
                (x) => x.oneofKind === `inline`,
                (inline) => {
                  return (
                    <OPAEditor
                      exp={inline.inline}
                      onChange={(v) => {
                        inline.inline = v;

                        updateReq();
                      }}
                    />
                  );
                },
              )
              .otherwise(() => <></>);
          },
        )
        .otherwise(() => (
          <></>
        ))}
    </div>
  );
};

export default Cond;
